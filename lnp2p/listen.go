package lnp2p

import (
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/logging"
)

type listeningthread struct {
	listener *lndc.Listener
}

func acceptConnections(listener *lndc.Listener, listenAddr string, pm *PeerManager) {

	// Set this up in-advance.
	stopEvent := &StopListeningPortEvent{
		ListenAddr: listenAddr,
		Reason:     "panic",
	}

	// Do this now in case we panic so we can do cleanup.
	defer publishStopEvent(stopEvent, pm.ebus)

	// Actually start listening for connections.
	for {

		netConn, err := listener.Accept()
		if err != nil {
			if err.Error() != "EOF" {
				logging.Infof("error accepting connections, exiting: %s\n", err.Error())
				break // usually means the socket was closed
			} else {
				logging.Debugf("got EOF on accepting connection, ignoring...\n")
				continue // the testing framework generates EOFs, this is fine
			}
		}

		lndcConn, ok := netConn.(*lndc.Conn)
		if !ok {
			// this should never happen
			logging.Errorf("didn't get an lndc connection from listener, wtf?\n")
			netConn.Close()
			continue
		}

		rpk := pubkey(lndcConn.RemotePub())
		rlitaddr := convertPubkeyToLitAddr(rpk)
		rnetaddr := lndcConn.RemoteAddr()

		logging.Infof("New connection from %s at %s\n", rlitaddr, rnetaddr.String())

		// Read the peer info from the DB.
		pi, err := pm.peerdb.GetPeerInfo(rlitaddr)
		if err != nil {
			logging.Warnf("problem loading peer info in DB (maybe this is ok?): %s\n", err.Error())
			netConn.Close()
			continue
		}

		// Add the peer data to the DB if we don't have it.
		if pi == nil {
			pi = &lncore.PeerInfo{
				LnAddr:   &rlitaddr,
				Nickname: nil,
				NetAddr:  rnetaddr.String(),
			}
			err = pm.peerdb.AddPeer(rlitaddr, *pi)
			if err != nil {
				// don't close it, I guess
				logging.Errorf("problem saving peer info to DB: %s\n", err.Error())
			}
		} else {
			// Idk yet?
		}

		// Create the actual peer object.
		npeer := &Peer{
			lnaddr:   rlitaddr,
			nickname: pi.Nickname,
			conn:     lndcConn,
			idpubkey: rpk,
			idx:      nil,
		}

		// Don't do any locking here since registerPeer takes a lock and Go's
		// mutex isn't reentrant.
		pm.registerPeer(npeer)

		// Start a goroutine to process inbound traffic for this peer.
		go processConnectionInboundTraffic(npeer, pm)

	}

	// Update the stop reason.
	stopEvent.Reason = "closed"

	// Then delete the entry from listening ports.
	pm.mtx.Lock()
	delete(pm.listeningPorts, listenAddr)
	pm.mtx.Unlock()

	// after this the stop event will be published
	logging.Infof("Stopped listening on %s\n", listenAddr)

}

func processConnectionInboundTraffic(peer *Peer, pm *PeerManager) {

	// Set this up in-advance.
	dcEvent := &PeerDisconnectEvent{
		Peer:   peer,
		Reason: "panic",
	}

	// Do this now in case we panic so we can do cleanup.
	defer publishDisconnectEvent(dcEvent, pm.ebus)

	// TODO Have chanmgr deal with channels after peer connection brought up. (eventbus)

	for {

		// Make a buf and read into it.
		buf := make([]byte, 1<<24)
		n, err := peer.conn.Read(buf)
		if err != nil {
			logging.Warnf("Error reading from peer: %s\n", err.Error())
			peer.conn.Close()
			return
		}

		logging.Debugf("Got message of len %d from peer %s\n", n, peer.GetLnAddr())

		// Send to the message processor.
		err = pm.mproc.HandleMessage(peer, buf[:n])
		if err != nil {
			logging.Errorf("Error proccessing message: %s\n", err.Error())
		}

	}

}

func publishStopEvent(event *StopListeningPortEvent, ebus *eventbus.EventBus) {
	ebus.Publish(*event)
}

func publishDisconnectEvent(event *PeerDisconnectEvent, ebus *eventbus.EventBus) {
	ebus.Publish(*event)
}
