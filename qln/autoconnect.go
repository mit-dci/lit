package qln

import (
	"time"

	"github.com/mit-dci/lit/logging"

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
func (nd *LitNode) AutoReconnect(port int, interval int64, connectedCoinOnly bool) {
	// Listen myself after a timeout
	_, err := nd.TCPListener(port)
	if err != nil {
		logging.Errorf("Could not start listening automatically: %s", err.Error())
		return
	}

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

	isConnectedCoin := func(peerIdx uint32) bool {
		// now only connect to those peers in the array
		for _, arr := range coinMap {
			for _, i := range arr {
				if peerIdx == i {
					return true
				}
			}
		}
		return false
	}

	var empty [33]byte
	i := uint32(1)
	for {

		pubKey, _ := nd.GetPubHostFromPeerIdx(i)
		if pubKey == empty {
			logging.Infof("Done, tried %d hosts\n", i-1)
			break
		}

		// If we're only reconnecting to peers we have channels with
		// in a connected coin type (daemon is available), then skip
		// peers that are not in that list
		if connectedCoinOnly && !isConnectedCoin(i) {
			logging.Infof("Skipping peer %d due to onlyConnectedCoins=true\n", i)
			i++
			continue
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
			err := nd.DialPeer(adr)
			if err != nil {
				logging.Errorf("Could not restore connection to %s: %s\n", adr, err.Error())
			}
			<-ticker.C
		}()

		i++
	}

}
