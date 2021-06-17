package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/MarinX/monerorpc"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	_ "github.com/lib/pq"
)

func getPosts(uri string, limit int) ([]map[string]interface{}, error) {
	method := "/api/v1/posts"
	u, _ := url.ParseRequestURI(uri)
	u.Path = method
	urlStr := u.String()
	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodGet, urlStr, bytes.NewBuffer([]byte("")))

	q := r.URL.Query()
	q.Add("view", "recent")
	q.Add("limit", string(limit))

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

func comment(uri string, apiKey string, address string) error {
	return nil
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
	latestPostNum := posts[0]["number"]

	defer db.Close()

	// determine the post_id that was last processed
	rows, err := db.Query(`
			SELECT scanned_up_to_id 
			FROM scan_progress
			WHERE type="post"`)
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

	numberOfUnprocessedPosts := latestPostNum - scannedUpToId

	posts, err = getPosts(uri, numberOfUnprocessedPosts)

	// need monero client and wallet to generate new addresses
	client := monerorpc.New(monerorpc.MainnetURI, nil)

	for i, post := range posts {
		bountyFundingAddress := client.Wallet.CreateAddress()
		comment(uri, apiKey, bountyFundingAddress)
	}

}
