package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
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

func comment(uri string, postNum int, apiKey string, subAddrBalance int, paymentAmt int) (interface{}, error) {
	method := "/api/v1/posts/" + strconv.Itoa(postNum) + "/comments"
	strData := `{"content":"Bounty increased by ` + strconv.Itoa(paymentAmt) + ` XMR\n Total Bounty: ` + strconv.Itoa(subAddrBalance) + ` XMR"}`
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

	var incomingTransfersReq wallet.IncomingTransfersRequest
	var incomingTransfersResp *wallet.IncomingTransfersResponse
	txToBeProcessed := new(wallet.IncomingTransfer)
	txToBeProcessed.TxHash = txId
	bountyTotal := float64(0)

	incomingTransfersReq.AccountIndex = uint64(0)
	incomingTransfersReq.TransferType = "all"

	client := monerorpc.New(monerorpc.TestnetURI, nil)
	incomingTransfersResp, err = client.Wallet.IncomingTransfers(&incomingTransfersReq)
	if err != nil {
		log.Println(err)
	}

	// there's no way to request information for a specific transaction
	// have to request incoming transfer, then loop to find the specific transaction

	for _, transfer := range incomingTransfersResp.Transfers {
		if transfer.TxHash == txToBeProcessed.TxHash {
			txToBeProcessed.Amount = transfer.Amount
			txToBeProcessed.GlobalIndex = transfer.GlobalIndex
			txToBeProcessed.KeyImage = transfer.KeyImage
			txToBeProcessed.Spent = transfer.Spent
			txToBeProcessed.SubaddrIndex = transfer.SubaddrIndex
			txToBeProcessed.TxSize = transfer.TxSize
		}
	}

	// convert piconero to monero
	paymentAmt := float64(txToBeProcessed.Amount) * float64(.000000000001)

	// only comment for payments that are above a certain threshold.
	// this is to avoid micropayment spam that would appear in the comments

	if paymentAmt > float64(.005) {
		// determine the post id that is associated with the incoming payment
		rows, err := db.Query(`
			SELECT post_number
			FROM post_address_mapping
			WHERE account_index = $1
			  AND address_index = $2`,
			incomingTransfersReq.AccountIndex,
			txToBeProcessed.SubaddrIndex.Minor)
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

		// determine the bounty total to be included in the comments
		for _, transfer := range incomingTransfersResp.Transfers {
			if transfer.SubaddrIndex == txToBeProcessed.SubaddrIndex {
				bountyTotal += float64(transfer.Amount) * float64(.000000000001)
			}
		}

		_, err = comment(uri, postNum, apiKey, int(bountyTotal), int(paymentAmt))
		if err != nil {
			log.Println(err)
		}
	}
}
