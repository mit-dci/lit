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
	nd.ChannelMap = make(map[[20]byte][]LinkDesc)

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
			theirLnAdr := bech32.Encode("ln", channel.Link.BPKH[:])
			if !graph.IsNode(theirLnAdr) {
				graph.AddNode("Lit", theirLnAdr, nil)
			}

			attrs := make(map[string]string)

			switch channel.Link.CoinType {
			case 0:
				attrs["color"] = "orange"
			case 28:
				attrs["color"] = "green"
			}

			attrs["label"] = strconv.FormatUint(uint64(channel.Link.CoinType), 10)

			graph.AddEdge(lnAdr, theirLnAdr, true, attrs)
		}
	}

	return "di" + graph.String()
}

func (nd *LitNode) FindPath(targetPkh [20]byte, destCoinType uint32, originCoinType uint32, amount int64, fee int64) ([]lnutil.RouteHop, error) {
	var myIdPkh [20]byte
	idHash := fastsha256.Sum256(nd.IdKey().PubKey().SerializeCompressed())
	copy(myIdPkh[:], idHash[:20])

	distance := make(map[lnutil.RouteHop]nodeWithDist)
	var nodeHeap distanceHeap

	nd.ChannelMapMtx.Lock()
	defer nd.ChannelMapMtx.Unlock()

	nwd := nodeWithDist{
		Dist:     0,
		Pkh:      myIdPkh,
		CoinType: originCoinType,
		Amt:      amount,
	}
	distance[lnutil.RouteHop{myIdPkh, originCoinType}] = nwd
	heap.Push(&nodeHeap, nwd)

	prev := make(map[lnutil.RouteHop]lnutil.RouteHop)

	for nodeHeap.Len() != 0 {
		partialPath := heap.Pop(&nodeHeap).(nodeWithDist)
		bestNode := partialPath.Pkh

		route := []lnutil.RouteHop{{bestNode, partialPath.CoinType}}
		for !(bytes.Equal(route[0].Node[:], myIdPkh[:]) && originCoinType == route[0].CoinType) {
			route = append([]lnutil.RouteHop{prev[route[0]]}, route...)
		}

		fmt.Print("Analyzing route: ")
		for _, node := range route {
			fmt.Printf("-> %s:%d", bech32.Encode("ln", node.Node[:]), node.CoinType)
		}
		fmt.Print("\n")

		if bytes.Equal(bestNode[:], targetPkh[:]) && partialPath.CoinType == destCoinType {
			break
		}

		fmt.Printf("Finding edges for %s...\n", bech32.Encode("ln", bestNode[:]))
		for _, channel := range nd.ChannelMap[bestNode] {
			fmt.Printf("Checking %s:%d\n", bech32.Encode("ln", channel.Link.BPKH[:]), channel.Link.CoinType)

			var rd *lnutil.RateDesc

			// do we need to exchange?
			// is the last hop coin type the same as this one?
			if partialPath.CoinType != channel.Link.CoinType {
				// we need to exchange, but is it possible?
				var rates []lnutil.RateDesc
				for _, link := range nd.ChannelMap[bestNode] {
					if link.Link.CoinType == partialPath.CoinType {
						rates = link.Link.Rates
						break
					}
				}

				for _, rate := range rates {
					if rate.CoinType == channel.Link.CoinType && rate.Rate > 0 {
						rd = &rate
						break
					}
				}

				// it's not possible to exchange these two coin types
				fmt.Printf("can't exchange %d for %d via %s", partialPath.CoinType, channel.Link.CoinType, bech32.Encode("ln", channel.Link.BPKH[:]))
				continue
			}

			amtRqd := partialPath.Amt

			// We need to exchange for this hop
			if rd != nil {
				// required capacity is last hop amt * rate
				if rd.Reciprocal {
					// prior hop coin type is worth less than this one
					amtRqd = partialPath.Amt / rd.Rate
				} else {
					// prior hop coin type is worth more than this one
					amtRqd = partialPath.Amt * rd.Rate
				}
			}

			if amtRqd < consts.MinOutput+fee {
				// exchanging to this point has pushed the amount too low
				fmt.Printf("exchanging %d for %d via %s pushes the amount too low: %d", partialPath.CoinType, channel.Link.CoinType, bech32.Encode("ln", channel.Link.BPKH[:]), amtRqd)
				continue
			}

			capOk := (channel.Link.ACapacity >= amtRqd)
			isTarget := bytes.Equal(targetPkh[:], channel.Link.BPKH[:])
			coinTypeMatch := (destCoinType == channel.Link.CoinType)

			fmt.Printf("Capok: [%t] - isTarget: [%t] - coinTypeMatch [%t] - dirty [%t]\n", capOk, isTarget, coinTypeMatch, channel.Dirty)

			if !channel.Dirty && capOk && (!isTarget || (isTarget && coinTypeMatch)) {

				tempDist := partialPath.Dist + 1
				dist, exists := distance[lnutil.RouteHop{channel.Link.BPKH, channel.Link.CoinType}]

				var prevDist int64
				if exists {
					prevDist = dist.Dist
				}

				if !exists || (exists && tempDist < prevDist) {
					// We could use this. Explore further

					newDist := nodeWithDist{
						Dist:     tempDist,
						Pkh:      channel.Link.BPKH,
						CoinType: channel.Link.CoinType,
						Amt:      amtRqd,
					}

					distance[lnutil.RouteHop{channel.Link.BPKH, channel.Link.CoinType}] = newDist

					prev[lnutil.RouteHop{channel.Link.BPKH, channel.Link.CoinType}] = lnutil.RouteHop{partialPath.Pkh, partialPath.CoinType}

					fmt.Printf("Pushing %s:%d onto heap\n", bech32.Encode("ln", channel.Link.BPKH[:]), channel.Link.CoinType)

					heap.Push(&nodeHeap, newDist)
				}
			}
		}
	}

	if _, ok := prev[lnutil.RouteHop{targetPkh, destCoinType}]; !ok {
		return nil, fmt.Errorf("No route to target %s:%d", bech32.Encode("ln", targetPkh[:]), destCoinType)
	}

	route := []lnutil.RouteHop{prev[lnutil.RouteHop{targetPkh, destCoinType}], {targetPkh, destCoinType}}
	for !(bytes.Equal(route[0].Node[:], myIdPkh[:]) && route[0].CoinType == originCoinType) {
		route = append([]lnutil.RouteHop{prev[route[0]]}, route...)
	}

	return route, nil
}

func (nd *LitNode) cleanStaleChannels() {
	nd.ChannelMapMtx.Lock()
	defer nd.ChannelMapMtx.Unlock()

	newChannelMap := make(map[[20]byte][]LinkDesc)

	now := time.Now().Unix()

	for pkh, node := range nd.ChannelMap {
		for _, channel := range node {
			if channel.Link.Timestamp+consts.ChannelAdvTimeout >= now {
				newChannelMap[pkh] = append(newChannelMap[pkh], channel)
			}
		}
	}

	nd.ChannelMap = newChannelMap
}

func (nd *LitNode) advertiseLinks(seq uint32) {
	caps := make(map[[20]byte]map[uint32]int64)

	nd.RemoteMtx.Lock()
	for _, peer := range nd.RemoteCons {
		for _, q := range peer.QCs {
			if !q.CloseData.Closed && q.State.MyAmt >= 2*(consts.MinOutput+q.State.Fee) && !q.State.Failed {
				outHash := fastsha256.Sum256(peer.Con.RemotePub().SerializeCompressed())
				var BPKH [20]byte
				copy(BPKH[:], outHash[:20])

				if _, ok := caps[BPKH]; !ok {
					caps[BPKH] = make(map[uint32]int64)
				}

				caps[BPKH][q.Coin()] += q.State.MyAmt - consts.MinOutput - q.State.Fee
			}
		}
	}

	nd.RemoteMtx.Unlock()

	var msgs []lnutil.LinkMsg

	outHash := fastsha256.Sum256(nd.IdKey().PubKey().SerializeCompressed())
	var APKH [20]byte
	copy(APKH[:], outHash[:20])

	for BPKH, node := range caps {
		for coin, capacity := range node {
			var outmsg lnutil.LinkMsg
			outmsg.CoinType = coin
			outmsg.Seq = seq

			outmsg.APKH = APKH
			outmsg.BPKH = BPKH

			outmsg.ACapacity = capacity

			outmsg.PeerIdx = math.MaxUint32

			msgs = append(msgs, outmsg)
		}
	}

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
			if bytes.Equal(v.Link.BPKH[:], msg.BPKH[:]) && v.Link.CoinType == msg.CoinType {
				// This is the link we've been looking for
				if msg.Seq <= v.Link.Seq {
					// Old advert
					return
				}

				// Update channel map
				nd.ChannelMap[msg.APKH][i].Link = msg
				nd.ChannelMap[msg.APKH][i].Dirty = false

				newChan = false
				break
			}
		}
	}

	if newChan {
		// New peer or new channel
		nd.ChannelMap[msg.APKH] = append(nd.ChannelMap[msg.APKH], LinkDesc{msg, false})
	}

	// Rebroadcast
	origIdx := msg.PeerIdx

	for peerIdx := range nd.RemoteCons {
		if peerIdx != origIdx {
			msg.PeerIdx = peerIdx

			go func(msg lnutil.LinkMsg) {
				nd.OmniOut <- msg
			}(msg)
		}
	}
}
