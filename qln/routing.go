package qln

import (
	"bytes"
	"container/heap"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/awalterschulze/gographviz"
	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
)

func (nd *LitNode) InitRouting() {
	nd.ChannelMapMtx.Lock()
	defer nd.ChannelMapMtx.Unlock()
	nd.ChannelMap = make(map[[20]byte][]LinkDesc)
	nd.ExchangeRates = make(map[uint32][]lnutil.RateDesc)

	err := nd.PopulateRates()
	if err != nil {
		logging.Errorf("failure loading exchange rates: %s", err.Error())
	}

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

// FindPath uses Bellman-Ford and Dijkstra to find the path with the best price that has enough capacity to route the payment
func (nd *LitNode) FindPath(targetPkh [20]byte, destCoinType uint32, originCoinType uint32, amount int64) ([]lnutil.RouteHop, error) {
	var myIdPkh [20]byte
	idHash := fastsha256.Sum256(nd.IdKey().PubKey().SerializeCompressed())
	copy(myIdPkh[:], idHash[:20])

	type routeHop struct {
		Node     [20]byte
		CoinType uint32
		Terminus bool
	}

	type channelEdge struct {
		W        float64
		U        routeHop
		V        routeHop
		Rate     lnutil.RateDesc
		Capacity int64
	}

	type channelEdgeLight struct {
		W        float64
		U        int
		V        int
		Rate     lnutil.RateDesc
		Capacity int64
	}

	// set up initial graph
	var edges []channelEdge
	var vertices []routeHop
	var edgesLight []channelEdgeLight

	verticesMap := make(map[routeHop]int)

	nd.ChannelMapMtx.Lock()

	// for each node visit the nodes connected via its channels and add a
	// BPKH:channel_cointype vertex to the graph for each of its channels.
	// Then for the current node (APKH), for each channel add an edge from
	// APKH:channel_cointype to each BPKH:cointype pair in the graph.

	for pkh, channels := range nd.ChannelMap {
		logging.Debugf("processing channels from %s", bech32.Encode("ln", pkh[:]))

		for _, channel := range channels {
			logging.Debugf("...processing channel %s:%d", bech32.Encode("ln", channel.Link.BPKH[:]), channel.Link.CoinType)
			var newEdges []channelEdge
			origin := routeHop{
				channel.Link.APKH,
				channel.Link.CoinType,
				false,
			}
			verticesMap[origin] = -1

			coinTypes := map[uint32]bool{}

			for _, theirChannel := range nd.ChannelMap[channel.Link.BPKH] {
				logging.Debugf("......checking outbound connection %s:%d", bech32.Encode("ln", theirChannel.Link.BPKH[:]), theirChannel.Link.CoinType)
				if _, ok := coinTypes[theirChannel.Link.CoinType]; !ok {
					var rd *lnutil.RateDesc
					for _, rate := range theirChannel.Link.Rates {
						if rate.CoinType == channel.Link.CoinType {
							rd = &rate
							break
						}
					}

					if rd == nil {
						// this trade is not possible
						logging.Debugf(".........ignoring channel because trade %d->%d is not possible", channel.Link.CoinType, theirChannel.Link.CoinType)
						continue
					}

					vertex := routeHop{
						channel.Link.BPKH,
						theirChannel.Link.CoinType,
						false,
					}
					verticesMap[vertex] = -1

					var price float64
					if rd.Reciprocal {
						price = 1.0 / float64(rd.Rate)
					} else {
						price = float64(rd.Rate)
					}

					weight := -math.Log(price)

					edge := channelEdge{
						weight,
						origin,
						vertex,
						*rd,
						channel.Link.ACapacity,
					}

					logging.Debugf(".........adding edge: %s:%d->%s:%d", bech32.Encode("ln", edge.U.Node[:]), edge.U.CoinType, bech32.Encode("ln", edge.V.Node[:]), edge.V.CoinType)

					newEdges = append(newEdges, edge)
					coinTypes[theirChannel.Link.CoinType] = true
				} else {
					logging.Debugf(".........ignoring channel because its cointype has already been covered")
				}
			}

			vertex := routeHop{
				channel.Link.BPKH,
				channel.Link.CoinType,
				true,
			}
			verticesMap[vertex] = -1

			edge := channelEdge{
				0,
				origin,
				vertex,
				lnutil.RateDesc{
					channel.Link.CoinType,
					1,
					false,
				},
				channel.Link.ACapacity,
			}

			logging.Debugf("...adding sink: %s:%d->%s:%d", bech32.Encode("ln", edge.U.Node[:]), edge.U.CoinType, bech32.Encode("ln", edge.V.Node[:]), edge.V.CoinType)

			newEdges = append(newEdges, edge)

			edges = append(edges, newEdges...)
		}
	}
	nd.ChannelMapMtx.Unlock()

	var predecessor []int
	var distance []float64

	for k, _ := range verticesMap {
		vertices = append(vertices, k)
		distance = append(distance, math.MaxFloat64)
		predecessor = append(predecessor, -1)
		verticesMap[k] = len(vertices) - 1
		logging.Debugf("vertex %x:%d: %d", k.Node, k.CoinType, len(vertices)-1)
	}

	for _, edge := range edges {
		edgesLight = append(edgesLight, channelEdgeLight{
			edge.W,
			verticesMap[edge.U],
			verticesMap[edge.V],
			edge.Rate,
			edge.Capacity,
		})
		U := vertices[edgesLight[len(edgesLight)-1].U]
		V := vertices[edgesLight[len(edgesLight)-1].V]
		logging.Debugf("adding edgeLight: %x:%d->%x:%d", U.Node, U.CoinType, V.Node, V.CoinType)
	}

	graph := gographviz.NewGraph()
	graph.SetName("\"bf-step\"")
	for _, edge := range edgesLight {
		U := fmt.Sprintf("\"%s:%d\"", bech32.Encode("ln", vertices[edge.U].Node[:]), vertices[edge.U].CoinType)
		V := fmt.Sprintf("\"%s:%d\"", bech32.Encode("ln", vertices[edge.V].Node[:]), vertices[edge.V].CoinType)

		nodeAttrs := make(map[string]string)
		if vertices[edge.V].Terminus {
			nodeAttrs["peripheries"] = "2"
			V = fmt.Sprintf("\"%s:%d terminus\"", bech32.Encode("ln", vertices[edge.V].Node[:]), vertices[edge.V].CoinType)
		}

		if !graph.IsNode(U) {
			graph.AddNode("\"bf-step\"", U, nil)
		}
		if !graph.IsNode(V) {
			graph.AddNode("\"bf-step\"", V, nodeAttrs)
		}

		attrs := make(map[string]string)

		attrs["label"] = fmt.Sprintf("\"weight: %f64, trade: %d->%d\"", edge.W, vertices[edge.U].CoinType, vertices[edge.V].CoinType)

		graph.AddEdge(U, V, true, attrs)
	}

	logging.Debugf("bf-step graph: \n %s", "di"+graph.String())

	// find my ID in map
	myId, ok := verticesMap[routeHop{myIdPkh, originCoinType, false}]
	if !ok {
		return nil, fmt.Errorf("origin node not found")
	}

	targetId, ok := verticesMap[routeHop{targetPkh, destCoinType, true}]
	if !ok {
		return nil, fmt.Errorf("destination node not found")
	}

	// add dummy vertex q to the map
	vertices = append(vertices, routeHop{[20]byte{}, 0, false})
	distance = append(distance, 0)
	predecessor = append(predecessor, -1)

	// connect q to every other vertex
	for idx := range vertices {
		if idx < len(vertices)-1 {
			edgesLight = append(edgesLight, channelEdgeLight{
				0,
				len(vertices) - 1,
				idx,
				lnutil.RateDesc{},
				0,
			})
		}
	}

	// relax the edges from q
	for i := 0; i < len(vertices); i++ {
		var relaxed bool
		for _, edge := range edgesLight {
			if distance[edge.U]+edge.W < distance[edge.V] {
				distance[edge.V] = distance[edge.U] + edge.W
				predecessor[edge.V] = edge.U
				relaxed = true
			}
		}

		// we didn't relax any edges in the last round so we can quit early
		if !relaxed {
			break
		}
	}

	// check for negative-weight cycles
	for _, edge := range edgesLight {
		if distance[edge.U]+edge.W < distance[edge.V] {
			return nil, fmt.Errorf("negative weight cycle in channel graph")
		}
	}

	// reweight original graph
	for idx, edge := range edgesLight {
		edgesLight[idx].W += distance[edge.U] - distance[edge.V]
	}

	// remove q and its edges
	edgesLight = edgesLight[:len(edges)]
	vertices = vertices[:len(vertices)-1]
	predecessor = predecessor[:len(predecessor)-1]
	distance = distance[:len(distance)-1]

	// run dijkstra over the reweighted graph to find the lowest weight route
	// with enough capacity to route the amount we want to send
	dDistance := make([]*nodeWithDist, len(vertices))
	dEdges := make([][]channelEdgeLight, len(vertices))

	dDistance[myId] = &nodeWithDist{
		0,
		myId,
		amount,
	}

	for idx := range predecessor {
		predecessor[idx] = -1
	}

	for _, edge := range edgesLight {
		dEdges[edge.U] = append(dEdges[edge.U], edge)
	}

	var nodeHeap distanceHeap

	heap.Push(&nodeHeap, *dDistance[myId])

	for nodeHeap.Len() > 0 {
		partialPath := heap.Pop(&nodeHeap).(nodeWithDist)

		logging.Debugf("popped %s:%d from heap", bech32.Encode("ln", vertices[partialPath.Node].Node[:]), vertices[partialPath.Node].CoinType)

		p, ok := coinparam.RegisteredNets[vertices[partialPath.Node].CoinType]
		if !ok {
			logging.Debugf("ignoring %s:%d because cointype is unknown", bech32.Encode("ln", vertices[partialPath.Node].Node[:]), vertices[partialPath.Node].CoinType)
			continue
		}

		fee := p.FeePerByte * 1000

		for _, edge := range dEdges[partialPath.Node] {
			logging.Debugf("considering edge %s:%d", bech32.Encode("ln", vertices[edge.V].Node[:]), vertices[edge.V].CoinType)

			amtRqd := partialPath.Amt

			if amtRqd < consts.MinOutput+fee {
				// this amount is too small to route
				logging.Debugf("ignoring %x:%d->%x:%d because amount rqd: %d less than minOutput+fee: %d", vertices[edge.U].Node, vertices[edge.U].CoinType, vertices[edge.V].Node, vertices[edge.V].CoinType, amtRqd, consts.MinOutput+fee)
				continue
			}

			if amtRqd > edge.Capacity {
				// this channel doesn't have enough capacity
				logging.Debugf("ignoring %x:%d->%x:%d because amount rqd: %d less than capacity: %d", vertices[edge.U].Node, vertices[edge.U].CoinType, vertices[edge.V].Node, vertices[edge.V].CoinType, amtRqd, edge.Capacity)
				continue
			}

			nextAmt := amtRqd

			// required capacity for next hop is last hop amt * rate
			if edge.Rate.Reciprocal {
				// prior hop coin type is worth less than this one
				nextAmt /= edge.Rate.Rate
			} else {
				// prior hop coin type is worth more than this one
				nextAmt *= edge.Rate.Rate
			}

			alt := dDistance[partialPath.Node].Dist + edge.W
			if dDistance[edge.V] == nil {
				dDistance[edge.V] = &nodeWithDist{
					alt,
					edge.V,
					nextAmt,
				}
			} else if alt < dDistance[edge.V].Dist {
				dDistance[edge.V].Dist = alt
				dDistance[edge.V].Amt = nextAmt
			} else {
				continue
			}

			logging.Debugf("could use edge %s:%d->%s:%d amt: %d capacity: %d", bech32.Encode("ln", vertices[edge.U].Node[:]), vertices[edge.U].CoinType, bech32.Encode("ln", vertices[edge.V].Node[:]), vertices[edge.V].CoinType, amtRqd, edge.Capacity)

			predecessor[edge.V] = edge.U
			heap.Push(&nodeHeap, *dDistance[edge.V])
		}
	}

	for target, dist := range dDistance {
		if dist != nil && vertices[target].Terminus {
			price := math.Exp(-dist.Dist)
			logging.Debugf("%s:%d: cap (recv): %d, price: %f64, cap (send): %d", bech32.Encode("ln", vertices[target].Node[:]), vertices[target].CoinType, dist.Amt, price, int(float64(dist.Amt)/price))
		}
	}

	if dDistance[targetId] == nil {
		return nil, fmt.Errorf("no route from origin to destination could be found")
	}

	routeIds := []int{predecessor[targetId], targetId}
	for predecessor[routeIds[0]] != -1 {
		routeIds = append([]int{predecessor[routeIds[0]]}, routeIds...)
	}

	var route []lnutil.RouteHop
	for _, id := range routeIds {
		route = append(route, lnutil.RouteHop{vertices[id].Node, vertices[id].CoinType})
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

				amt := q.State.MyAmt - consts.MinOutput - q.State.Fee

				// Since we don't yet perform multi-path routing, the capacity
				// is the maximum available single channel capacity
				if caps[BPKH][q.Coin()] < amt {
					caps[BPKH][q.Coin()] = amt
				}
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

			if rates, ok := nd.ExchangeRates[coin]; ok {
				outmsg.Rates = rates
			}

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

			go func(omsg lnutil.LinkMsg) {
				nd.tmpSendLitMsg(omsg)
			}(msg)
		}
	}
}

func (nd *LitNode) PopulateRates() error {
	ratesPath := filepath.Join(nd.LitFolder, "rates.json")

	jsonFile, err := os.Open(ratesPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		logging.Infof("Rates file not found.")
		return nil
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}
	err = json.Unmarshal(byteValue, &nd.ExchangeRates)
	if err != nil {
		return err
	}

	return nil
}
