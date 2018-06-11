package qln

import (
	"bytes"
	"fmt"
	"time"

	"github.com/adiabat/bech32"
	"github.com/btcsuite/fastsha256"
)

// AutoReconnect will start listening for incoming connections
// and attempt to automatically reconnect to all
// previously known peers.
func (nd *LitNode) AutoReconnect() {
	// Listen myself
	// TODO : configurable port for this?
	nd.TCPListener(":2448")

	// Reconnect to other nodes after a timeout
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			var empty [33]byte
			i := uint32(0)
			pubKey, _ := nd.GetPubHostFromPeerIdx(i)
			for {
				alreadyConnected := false

				nd.RemoteMtx.Lock()
				for _, con := range nd.RemoteCons {
					if bytes.Equal(con.Con.RemotePub.SerializeCompressed(), pubKey[:]) {
						alreadyConnected = true
					}
				}
				nd.RemoteMtx.Unlock()

				if alreadyConnected {
					continue
				}

				idHash := fastsha256.Sum256(pubKey[:])
				adr := bech32.Encode("ln", idHash[:20])

				err := nd.DialPeer(adr)

				if err != nil {
					fmt.Printf("Could not restore connection to %s: %s\n", adr, err.Error())
				}
				i++
				pubKey, _ = nd.GetPubHostFromPeerIdx(i)
				if pubKey == empty {
					break
				}
			}
		}
	}()

}
