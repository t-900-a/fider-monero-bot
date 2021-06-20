package main

import (
	"fmt"
	"github.com/MarinX/monerorpc"
	wallet "github.com/MarinX/monerorpc/wallet"
)

func main() {
	//r.Header.Add("Accept", "application/json")
	//r.Header.Add("Accept-Encoding", "gzip, deflate, br")
	//r.Header.Add("Content-Type", "application/json")
	//r.Header.Add("Authorization", "Bearer vS9U4vSrjVFfkZZ9Rp3RxPoqe8JXyrvvPOps4y2pLuu9w8HgSbsWBuJENREpfyFA")
	var addressReq wallet.CreateAddressRequest
	client := monerorpc.New(monerorpc.TestnetURI, nil)

	addressReq.AccountIndex = 0
	bountyFundingAddressResp, err := client.Wallet.CreateAddress(&addressReq)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(bountyFundingAddressResp.Address)
	fmt.Println(bountyFundingAddressResp.AddressIndex)

}
