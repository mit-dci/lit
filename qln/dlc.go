package qln

func (nd *LitNode) OfferDlc(
	peerIdx, cointype uint32, outAmount, theirAmount, valueAllOurs, valueAllTheirs uint64, oraclePub, oracleOts [33]byte) (uint32, error) {
	/*
		_, ok := nd.SubWallet[cointype]
		if !ok {
			return 0, fmt.Errorf("No wallet of type %d connected", cointype)
		}

		nd.InProg.mtx.Lock()
		//	defer nd.InProg.mtx.Lock()
		if nd.InProg.PeerIdx != 0 {
			nd.InProg.mtx.Unlock()
			return 0, fmt.Errorf("fund with peer %d not done yet", nd.InProg.PeerIdx)
		}

		// TODO - would be convenient if it auto connected to the peer huh
		if !nd.ConnectedToPeer(peerIdx) {
			nd.InProg.mtx.Unlock()
			return 0, fmt.Errorf("Not connected to peer %d. Do that yourself.", peerIdx)
		}

		cIdx, err := nd.NextChannelIdx()
		if err != nil {
			nd.InProg.mtx.Unlock()
			return 0, err
		}

		nd.InProg.ChanIdx = cIdx
		nd.InProg.PeerIdx = peerIdx
		nd.InProg.Amt = ccap
		nd.InProg.InitSend = initSend
		nd.InProg.Data = data

		nd.InProg.Coin = cointype
		nd.InProg.mtx.Unlock() // switch to defer

		outMsg := lnutil.NewDlcOfferMsg(peerIdx, cointype, ourAmount, theirAmount, valueAllOurs, valueAlltheirs,)

		nd.OmniOut <- outMsg

		// wait until it's done!
		idx := <-nd.InProg.done
		return idx, nil*/
	return 0, nil
}
