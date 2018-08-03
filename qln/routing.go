package qln

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"time"

	"container/heap"

	"github.com/awalterschulze/gographviz"
	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/lnutil"
)

func (nd *LitNode) InitRouting() {
	nd.ChannelMapMtx.Lock()
	defer nd.ChannelMapMtx.Unlock()
	nd.ChannelMap = make(map[[20]byte][]lnutil.LinkMsg)

	nd.AdvTimeout = time.NewTicker(15 * time.Second)

	go func() {
		seq := uint32(0)

		for {
			nd.cleanStaleChannels()
			nd.advertiseLinks(seq)
			seq++
			<-nd.AdvTimeout.C
		}
	}()
}

func (nd *LitNode) VisualiseGraph() string {
	graph := gographviz.NewGraph()
	graph.SetName("Lit")

	nd.ChannelMapMtx.Lock()
	defer nd.ChannelMapMtx.Unlock()

	for pkh, node := range nd.ChannelMap {
		lnAdr := bech32.Encode("ln", pkh[:])
		if !graph.IsNode(lnAdr) {
			graph.AddNode("Lit", lnAdr, nil)
		}

		for _, channel := range node {
			theirLnAdr := bech32.Encode("ln", channel.BPKH[:])
			if !graph.IsNode(theirLnAdr) {
				graph.AddNode("Lit", theirLnAdr, nil)
			}

			attrs := make(map[string]string)

			switch channel.CoinType {
			case 0:
				attrs["color"] = "orange"
			case 28:
				attrs["color"] = "green"
			}

			attrs["label"] = strconv.FormatUint(uint64(channel.CoinType), 10)

			graph.AddEdge(lnAdr, theirLnAdr, true, attrs)
		}
	}

	return "di" + graph.String()
}

func (nd *LitNode) FindPath(targetPkh [20]byte, coinType uint32, amount int64, fee int64) ([][20]byte, error) {

	var myIdPkh [20]byte
	idHash := fastsha256.Sum256(nd.IdKey().PubKey().SerializeCompressed())
	copy(myIdPkh[:], idHash[:20])

	distance := make(map[[20]byte]nodeWithDist)
	distance[myIdPkh] = nodeWithDist{
		Dist: 0,
		Pkh:  myIdPkh,
	}

	prev := make(map[[20]byte][20]byte)

	var nodeHeap distanceHeap
	heap.Push(&nodeHeap, distance[myIdPkh])

	for nodeHeap.Len() != 0 {
		partialPath := heap.Pop(&nodeHeap).(nodeWithDist)
		bestNode := partialPath.Pkh

		route := [][20]byte{bestNode}
		for !bytes.Equal(route[0][:], myIdPkh[:]) {
			route = append([][20]byte{prev[route[0]]}, route...)
		}

		fmt.Print("Analyzing route: ")
		for _, node := range route {
			fmt.Printf("-> %s", bech32.Encode("ln", node[:]))
		}
		fmt.Print("\n")

		if bytes.Equal(bestNode[:], targetPkh[:]) {
			break
		}

		fmt.Printf("Finding edges for %s...\n", bech32.Encode("ln", bestNode[:]))
		for _, channel := range nd.ChannelMap[bestNode] {
			fmt.Printf("Checking %s\n", bech32.Encode("ln", channel.BPKH[:]))
			capOk := (channel.ACapacity-consts.MinOutput-fee >= amount)
			isTarget := bytes.Equal(targetPkh[:], channel.BPKH[:])
			coinTypeMatch := (coinType == channel.CoinType)

			fmt.Printf("Capok: [%t] - isTarget: [%t] - coinTypeMatch [%t]\n", capOk, isTarget, coinTypeMatch)

			if capOk && (!isTarget || (isTarget && coinTypeMatch)) {

				tempDist := partialPath.Dist + 1
				_, exists := distance[channel.BPKH]

				if !exists || (exists && tempDist < distance[channel.BPKH].Dist) {
					// We could use this. Explore further

					distance[channel.BPKH] = nodeWithDist{
						Dist: tempDist,
						Pkh:  channel.BPKH,
					}

					prev[channel.BPKH] = bestNode

					fmt.Printf("Pushing %s onto heap\n", bech32.Encode("ln", channel.BPKH[:]))

					heap.Push(&nodeHeap, distance[channel.BPKH])
				}
			}
		}
	}

	if _, ok := prev[targetPkh]; !ok {
		return nil, fmt.Errorf("No route to target %x", bech32.Encode("ln", targetPkh[:]))
	}

	route := [][20]byte{prev[targetPkh], targetPkh}
	for !bytes.Equal(route[0][:], myIdPkh[:]) {
		route = append([][20]byte{prev[route[0]]}, route...)
	}
	return route, nil
}

func (nd *LitNode) cleanStaleChannels() {
	nd.ChannelMapMtx.Lock()
	defer nd.ChannelMapMtx.Unlock()

	newChannelMap := make(map[[20]byte][]lnutil.LinkMsg)

	now := time.Now().Unix()

	for pkh, node := range nd.ChannelMap {
		for _, channel := range node {
			if channel.Timestamp+consts.ChannelAdvTimeout >= now {
				newChannelMap[pkh] = append(newChannelMap[pkh], channel)
			}
		}
	}

	nd.ChannelMap = newChannelMap
}

func (nd *LitNode) advertiseLinks(seq uint32) {
	nd.RemoteMtx.Lock()

	var msgs []lnutil.LinkMsg

	for _, peer := range nd.RemoteCons {
		for _, q := range peer.QCs {
			if !q.CloseData.Closed && q.State.MyAmt > 2*(consts.MinOutput+q.State.Fee) && !q.State.Failed {
				var outmsg lnutil.LinkMsg
				outmsg.CoinType = q.Coin()
				outmsg.Seq = seq

				var idPub [33]byte
				copy(idPub[:], nd.IdKey().PubKey().SerializeCompressed())

				var theirIdPub [33]byte
				copy(theirIdPub[:], peer.Con.RemotePub().SerializeCompressed())

				outHash := fastsha256.Sum256(idPub[:])
				copy(outmsg.APKH[:], outHash[:20])

				outHash = fastsha256.Sum256(theirIdPub[:])
				copy(outmsg.BPKH[:], outHash[:20])

				outmsg.ACapacity = q.State.MyAmt
				copy(outmsg.PKHScript[:], q.Op.Hash.CloneBytes()[:20])

				outmsg.PeerIdx = math.MaxUint32

				msgs = append(msgs, outmsg)
			}
		}
	}

	nd.RemoteMtx.Unlock()

	for _, msg := range msgs {
		nd.LinkMsgHandler(msg)
	}
}

func (nd *LitNode) LinkMsgHandler(msg lnutil.LinkMsg) {
	nd.ChannelMapMtx.Lock()
	defer nd.ChannelMapMtx.Unlock()
	nd.RemoteMtx.Lock()
	defer nd.RemoteMtx.Unlock()

	msg.Timestamp = time.Now().Unix()
	newChan := true

	// Check if node exists as a router
	if _, ok := nd.ChannelMap[msg.APKH]; ok {
		// Check if link state is most recent (seq)
		for i, v := range nd.ChannelMap[msg.APKH] {
			if bytes.Compare(v.PKHScript[:], msg.PKHScript[:]) == 0 {
				// This is the link we've been looking for
				if msg.Seq <= v.Seq {
					// Old advert
					return
				}

				// Update channel map
				nd.ChannelMap[msg.APKH][i] = msg

				newChan = false
				break
			}
		}
	}

	if newChan {
		// New peer or new channel
		nd.ChannelMap[msg.APKH] = append(nd.ChannelMap[msg.APKH], msg)
	}

	// Rebroadcast
	origIdx := msg.PeerIdx

	for peerIdx, _ := range nd.RemoteCons {
		if peerIdx != origIdx {
			msg.PeerIdx = peerIdx

			go func(msg lnutil.LinkMsg) {
				nd.OmniOut <- msg
			}(msg)
		}
	}
}
