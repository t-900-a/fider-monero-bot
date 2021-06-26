package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MarinX/monerorpc"
	wallet "github.com/MarinX/monerorpc/wallet"
	qrcode "github.com/skip2/go-qrcode"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"

	_ "github.com/lib/pq"
)

var floatType = reflect.TypeOf(float64(0))

func getFloat(unk interface{}) (float64, error) {
	v := reflect.ValueOf(unk)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(floatType) {
		return 0, fmt.Errorf("cannot convert %v to float64", v.Type())
	}
	fv := v.Convert(floatType)
	return fv.Float(), nil
}

func getPosts(uri string, limit float64) ([]map[string]interface{}, error) {
	method := "/api/v1/posts"
	u, _ := url.ParseRequestURI(uri)
	u.Path = method
	urlStr := u.String()
	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodGet, urlStr, bytes.NewBuffer([]byte("")))

	q := r.URL.Query()
	q.Add("view", "recent")
	q.Add("limit", strconv.FormatFloat(limit, 'f', 0, 64))

	r.URL.RawQuery = q.Encode()

	resp, err := client.Do(r)
	if err != nil {
		log.Println(err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		defer resp.Body.Close()
		resBody, _ := ioutil.ReadAll(resp.Body)
		response := string(resBody)
		resBytes := []byte(response)
		var resArr []map[string]interface{}
		err = json.Unmarshal(resBytes, &resArr)
		if err != nil {
			log.Println(err)
		}
		return resArr, nil
	} else {
		return nil, fmt.Errorf(string(resp.StatusCode) + " when making call:" + urlStr)
	}
}

func comment(uri string, postNum float64, apiKey string, address *string) (float64, error) {
	var png []byte
	png, err := qrcode.Encode("monero:"+*address, qrcode.Medium, 200)
	method := "/api/v1/posts/" + strconv.FormatFloat(postNum, 'f', 0, 64) + "/comments"
	strData := `{"content":"Donate to the address below to fund this bounty \n` + *address + `\nYour donation will be reflected in the comments. \nPayouts will be made once the bounty is complete to the individual(s) who completed the bounty first.",
		"attachments":[{"upload":{"fileName":"post_` + strconv.FormatFloat(postNum, 'f', 0, 64) + `_qr.png","contentType":"image/png","content":"` + base64.StdEncoding.EncodeToString(png) + `"}}]}`
	jsonData := []byte(strData)
	u, _ := url.ParseRequestURI(uri)
	u.Path = method
	urlStr := u.String()
	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodPost, urlStr, bytes.NewBuffer([]byte(jsonData)))
	r.Header.Add("Authorization", "Bearer "+apiKey)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))

	resp, err := client.Do(r)
	if err != nil {
		log.Println(err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		defer resp.Body.Close()
		resBody, _ := ioutil.ReadAll(resp.Body)
		response := string(resBody)
		resBytes := []byte(response)
		var resVar map[string]interface{}
		err = json.Unmarshal(resBytes, &resVar)
		if err != nil {
			log.Println(err)
		}

		commentId, err := getFloat(resVar["id"])
		return commentId, err
	} else {
		return float64(0), errors.New("request " + strconv.Itoa(resp.StatusCode) + " for " + urlStr)
	}
}

func main() {
	// database connection string that should be passed to binary
	db, err := sql.Open("postgres", os.Args[1])
	if err != nil {
		panic(err)
	}
	// api-key argument that should be passed
	apiKey := os.Args[2]
	// https://bounties.getmonero.org
	uri := os.Args[3]

	// get the most recent post id
	posts, err := getPosts(uri, 1)
	latestPostNumf, err := getFloat(posts[0]["number"])
	if err != nil {
		log.Println(err)
	}

	defer db.Close()

	// determine the post_id that was last processed
	rows, err := db.Query(`
			SELECT scanned_up_to_id 
			FROM scan_progress
			WHERE type='post'`)
	if err != nil {
		panic(err)
	}

	var (
		scannedUpToId int
	)
	for rows.Next() {
		if err := rows.Scan(&scannedUpToId); err != nil {
			panic(err)
		}
	}

	numberOfUnprocessedPosts := latestPostNumf - float64(scannedUpToId)
	if numberOfUnprocessedPosts > float64(0) {
		posts, err = getPosts(uri, numberOfUnprocessedPosts)
		if err != nil {
			log.Println(err)
		}

		// need monero client and wallet to generate new addresses
		client := monerorpc.New(monerorpc.TestnetURI, nil)
		var postNumi int
		var addressReq wallet.CreateAddressRequest
		var address string
		var addressIndex int64

		for _, post := range posts {
			// generate new address to be associated with the post

			addressReq.AccountIndex = 0
			addressResp, err := client.Wallet.CreateAddress(&addressReq)

			address = addressResp.Address
			addressIndex = int64(addressResp.AddressIndex)

			if err != nil {
				log.Println(err)
			}
			postNumf, err := getFloat(post["number"])
			postNumi = int(postNumf)
			if err != nil {
				log.Println(err)
			}
			_, err = comment(uri, postNumf, apiKey, &address)
			if err != nil {
				log.Println(err)
			}

		}

		_, err := db.Exec(`
		UPDATE scan_progress
		SET scanned_up_to_id = $1
		WHERE type = 'post'
	`, int(postNumi))
		if err != nil {
			log.Println(err)
		}

		_, err = db.Exec(`
		INSERT INTO post_address_mapping (
		    post_number, account_index, address_index, subaddress 
		)
		VALUES (
		    $1, $2, $3, $4
		)
		`, &postNumi, addressReq.AccountIndex, &addressIndex, &address)

		if err != nil {
			log.Println(err)
		}

	}
}
