package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MarinX/monerorpc"
	wallet "github.com/MarinX/monerorpc/wallet"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	_ "github.com/lib/pq"
)

func comment(uri string, postNum int, apiKey string, subAddrBalance uint64, paymentAmt uint64) (interface{}, error) {
	method := "/api/v1/posts/" + strconv.Itoa(postNum) + "/comments"
	strData := `{"content":"Bounty increased by ` + strconv.Itoa(int(paymentAmt)) + ` XMR\n Total Bounty: ` + strconv.Itoa(int(subAddrBalance)) + `"}`
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

		return resVar["id"], err
	} else {
		return nil, errors.New("request " + strconv.Itoa(resp.StatusCode) + " for " + urlStr)
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
	//tx id that is passed by tx notify
	txId := os.Args[4]

	var paymentReq wallet.GetPaymentsRequest
	var paymentResp *wallet.GetPaymentsResponse

	paymentReq.PaymentID = txId

	client := monerorpc.New(monerorpc.TestnetURI, nil)
	fmt.Println(txId)
	paymentResp, err = client.Wallet.GetPayments(&paymentReq)
	fmt.Println(paymentResp)
	if err != nil {
		log.Println(err)
	}

	// only comment if the payment is above a certain amount, to prevent comment spam
	for _, payment := range paymentResp.Payments {
		fmt.Println("Amount")
		fmt.Println(payment.Amount)
		if float64(payment.Amount) > float64(.005) {
			// determine the post id that is associated with the incoming payment
			rows, err := db.Query(`
			SELECT post_number
			FROM post_address_mapping
			WHERE account_index = $1
			  AND address_index = $2`,
				payment.SubaddrIndex.Major,
				payment.SubaddrIndex.Minor)
			if err != nil {
				panic(err)
			}

			var (
				postNum int
			)
			for rows.Next() {
				if err := rows.Scan(&postNum); err != nil {
					panic(err)
				}
			}

			var balanceReq wallet.GetBalanceRequest
			var balanceResp *wallet.GetBalanceResponse

			balanceReq.AccountIndex = payment.SubaddrIndex.Major
			balanceReq.AddressIndices = payment.SubaddrIndex.Minor
			balanceResp, err = client.Wallet.GetBalance(&balanceReq)
			if err != nil {
				log.Println(err)
			}
			_, err = comment(uri, postNum, apiKey, balanceResp.Balance, payment.Amount)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
