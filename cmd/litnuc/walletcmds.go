package main

import (
	"fmt"

	"github.com/mit-dci/lit/litrpc"
)

// Address gets a new address from the lit node
func (lu *litNucClient) Address() (string, error) {

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
