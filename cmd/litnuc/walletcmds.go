package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/mit-dci/lit/litrpc"
)

// Address gets a new address from the lit node
func (lu *litNucClient) NewAddress() (string, error) {

	// cointype of 0 means default, not mainnet.
	// this is ugly but does prevent mainnet use for now.

	var cointype, numadrs uint32

	// if no arguments given, generate 1 new address.
	// if no cointype given, assume type 1 (testnet)

	numadrs = 1

	reply := new(litrpc.AddressReply)

	args := new(litrpc.AddressArgs)
	args.CoinType = cointype
	args.NumToMake = numadrs

	fmt.Printf("adr cointye: %d num:%d\n", args.CoinType, args.NumToMake)
	err := lu.rpccon.Call("LitRPC.Address", args, reply)
	if err != nil {
		return "", err
	}
	response := reply.WitAddresses[0]
	//	fmt.Fprintf(color.Output, "new adr(s): %s\nold: %s\n",
	//		lnutil.Address(reply.WitAddresses), lnutil.Address(reply.LegacyAddresses))
	return response, nil // reply.WitAddresses[]

}

// Address gets a new address from the lit node
func (lu *litNucClient) Address() (string, error) {

	// cointype of 0 means default, not mainnet.
	// this is ugly but does prevent mainnet use for now.

	var cointype, numadrs uint32

	// if no arguments given, generate 1 new address.
	// if no cointype given, assume type 1 (testnet)

	numadrs = 0

	reply := new(litrpc.AddressReply)

	args := new(litrpc.AddressArgs)
	args.CoinType = cointype
	args.NumToMake = numadrs

	fmt.Printf("adr cointye: %d num:%d\n", args.CoinType, args.NumToMake)
	err := lu.rpccon.Call("LitRPC.Address", args, reply)
	if err != nil {
		return "", err
	}
	response := reply.WitAddresses[len(reply.WitAddresses)-1]
	return response, nil
}

// Send sends coins somewhere
func (lu *litNucClient) Send(adr, amtString string) (string, error) {

	args := new(litrpc.SendArgs)
	reply := new(litrpc.TxidsReply)

	amt, err := strconv.Atoi(amtString)
	if err != nil {
		return "", err
	}

	log.Printf("send %d to address: %s \n", amt, adr)

	args.DestAddrs = []string{adr}
	args.Amts = []int64{int64(amt)}

	err = lu.rpccon.Call("LitRPC.Send", args, reply)
	if err != nil {
		return "", err
	}

	resp := fmt.Sprintf("sent txid(s):\n")
	for _, t := range reply.Txids {
		resp += fmt.Sprintf("%s\n", t)
	}
	return resp, nil
}
