package main

import (
	"fmt"

	"github.com/chzyer/readline"
	"github.com/mit-dci/lit/litrpc"
)

func (lc *litAfClient) completePeers(line string) []string {
	names := make([]string, 0)
	pReply := new(litrpc.ListConnectionsReply)
	err := lc.Call("LitRPC.ListConnections", nil, pReply)
	if err != nil {
		return names
	}
	if len(pReply.Connections) > 0 {
		for _, peer := range pReply.Connections {
			var peerStr = fmt.Sprint(peer.PeerNumber)
			names = append(names, peerStr)
		}
	}
	return names
}

func (lc *litAfClient) completeClosedPeers(line string) []string {
	channelpeers := make([]string, 0)
	connectedpeers := make([]string, 0)
	pReply := new(litrpc.ListConnectionsReply)
	cReply := new(litrpc.ChannelListReply)
	err := lc.Call("LitRPC.ListConnections", nil, pReply)
	if err != nil {
		return channelpeers
	}
	err = lc.Call("LitRPC.ChannelList", nil, cReply)
	if err != nil {
		return channelpeers
	}
	if len(pReply.Connections) > 0 {
		for _, peer := range pReply.Connections {
			var peerStr = fmt.Sprint(peer.PeerNumber)
			connectedpeers = append(connectedpeers, peerStr)
		}
	}
	for _, c := range cReply.Channels {
		var peerid = fmt.Sprint(c.PeerIdx)
		var found = false
		for _, v := range append(connectedpeers, channelpeers...) {
			if v == peerid {
				found = true
				break
			}
		}
		if !found {
			channelpeers = append(channelpeers, peerid)
		}
	}
	return channelpeers
}

func (lc *litAfClient) completeChannelIdx(line string) []string {
	names := make([]string, 0)
	cReply := new(litrpc.ChannelListReply)
	err := lc.Call("LitRPC.ChannelList", nil, cReply)
	if err != nil {
		return names
	}
	for _, c := range cReply.Channels {
		if !c.Closed {
			var cidxStr = fmt.Sprint(c.CIdx)
			names = append(names, cidxStr)
		}
	}
	return names
}

func (lc *litAfClient) NewAutoCompleter() readline.AutoCompleter {
	var completer = readline.NewPrefixCompleter(
		readline.PcItem("help",
			readline.PcItem("say"),
			readline.PcItem("ls"),
			readline.PcItem("con"),
			readline.PcItem("lis"),
			readline.PcItem("adr"),
			readline.PcItem("send"),
			readline.PcItem("fan"),
			readline.PcItem("sweep"),
			readline.PcItem("fund"),
			readline.PcItem("push"),
			readline.PcItem("close"),
			readline.PcItem("break"),
			readline.PcItem("stop"),
			readline.PcItem("exit"),
		),
		readline.PcItem("say",
			readline.PcItemDynamic(lc.completePeers)),
		readline.PcItem("ls"),
		readline.PcItem("con",
			readline.PcItemDynamic(lc.completeClosedPeers)),
		readline.PcItem("lis"),
		readline.PcItem("adr"),
		readline.PcItem("send"),
		readline.PcItem("fan"),
		readline.PcItem("sweep"),
		readline.PcItem("fund",
			readline.PcItemDynamic(lc.completePeers)),
		readline.PcItem("push",
			readline.PcItemDynamic(lc.completeChannelIdx)),
		readline.PcItem("close",
			readline.PcItemDynamic(lc.completeChannelIdx)),
		readline.PcItem("break",
			readline.PcItemDynamic(lc.completeChannelIdx)),
		readline.PcItem("stop"),
		readline.PcItem("exit"),
	)

	return completer
}
