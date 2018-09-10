package qln

import (
	"log"
	"time"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/crypto/fastsha256"
)

func removeDuplicates(inputArr []uint32) []uint32 {
	keys := make(map[uint32]bool)
	outputArr := []uint32{}
	for _, entry := range inputArr {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			outputArr = append(outputArr, entry)
		}
	}
	return outputArr
}

// AutoReconnect will start listening for incoming connections
// and attempt to automatically reconnect to all
// previously known peers attached with the coin daemons running.
func (nd *LitNode) AutoReconnect(listenPort string, interval int64) {
	// Listen myself after a timeout
	nd.TCPListener(listenPort)
	// Reconnect to other nodes after an interval
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	qcs, _ := nd.GetAllQchans() // get all chan data
	coinMap := make(map[uint32][]uint32)
	for _, qc := range qcs {
		// make a map of all channel data with the key as coinType
		coinMap[qc.Coin()] = append(coinMap[qc.Coin()], qc.KeyGen.Step[3]&0x7fffffff)
	}
	for i, arr := range coinMap {
		// remove duplicates in the map
		if nd.ConnectedCoinTypes[i] {
			coinMap[i] = removeDuplicates(arr)
		}
	}
	// now only connect to those peers in the array
	for j, arr := range coinMap {
		log.Printf("Trying to connect to %d Peers attached with coinType: %d", len(arr), j)
		for _, i := range arr {
			var empty [33]byte
			pubKey, _ := nd.GetPubHostFromPeerIdx(i)
			if pubKey == empty {
				log.Printf("Done, tried %d hosts\n", i-1)
				break
			}
			nd.RemoteMtx.Lock()
			_, alreadyConnected := nd.RemoteCons[i]
			nd.RemoteMtx.Unlock()
			if alreadyConnected {
				continue
			}
			idHash := fastsha256.Sum256(pubKey[:])
			adr := bech32.Encode("ln", idHash[:20])
			go func() {
				_, err := nd.DialPeer(adr)
				if err != nil {
					log.Printf("Could not restore connection to %s: %s\n", adr, err.Error())
				}
				<-ticker.C
			}()
		}
	}
}
