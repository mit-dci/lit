package lnp2p

import (
	"net"

	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lncore"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/logging"
)

type listeningthread struct {
	listener *lndc.Listener
}

func acceptConnections(listener *lndc.Listener, port int, pm *PeerManager) {

	// Set this up in-advance.
	stopEvent := &StopListeningPortEvent{
		Port:   port,
		Reason: "panic",
	}

	// Do this now in case we panic so we can do cleanup.
	defer publishStopEvent(stopEvent, pm.ebus)

	// Also make sure to remove this entry from the listening ports list later
	// on, regardless of how it gets removed.
	defer (func() {
		pm.mtx.Lock()
		delete(pm.listeningPorts, port)
		pm.mtx.Unlock()
	})()

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

		// Create the actual peer object.
		npeer := &Peer{
			lnaddr:   rlitaddr,
			nickname: nil,
			conn:     lndcConn,
			idpubkey: rpk,

			// TEMP
			idx: nil,
		}

		// Add the peer data to the DB if we don't have it.
		if pi == nil {
			raddr := rnetaddr.String()
			pidx, err := pm.peerdb.GetUniquePeerIdx()
			if err != nil {
				logging.Errorf("problem getting unique peeridx: %s\n", err.Error())
			}
			pi = &lncore.PeerInfo{
				LnAddr:   &rlitaddr,
				Nickname: nil,
				NetAddr:  &raddr,
				PeerIdx:  pidx,
			}
			err = pm.peerdb.AddPeer(rlitaddr, *pi)
			npeer.idx = &pidx
			if err != nil {
				// don't close it, I guess
				logging.Errorf("problem saving peer info to DB: %s\n", err.Error())
			}
		} else {
			npeer.nickname = pi.Nickname
			// TEMP
			npeer.idx = &pi.PeerIdx
		}

		// Don't do any locking here since registerPeer takes a lock and Go's
		// mutex isn't reentrant.
		pm.registerPeer(npeer)

		// Start a goroutine to process inbound traffic for this peer.
		go processConnectionInboundTraffic(npeer, pm)

	}

	// Update the stop reason.
	stopEvent.Reason = "closed"

	// after this the stop event will be published
	logging.Infof("Stopped listening on %s\n", port)

}

func processConnectionInboundTraffic(peer *Peer, pm *PeerManager) {

	// Set this up in-advance.
	dcEvent := &PeerDisconnectEvent{
		Peer:   peer,
		Reason: "panic",
	}

	// Do this now in case we panic so we can do cleanup.
	defer publishDisconnectEvent(dcEvent, pm.ebus)

	// And make sure to mark the peer object as dead when we leave here.
	defer (func() {
		peer.alive = false
	})()

	// TODO Have chanmgr deal with channels after peer connection brought up. (eventbus)

	for {

		// Make a buf and read into it.
		buf := make([]byte, 1<<24)

		// Actually read.
		n, err := peer.conn.Read(buf)
		if err != nil {
			logging.Warnf("Error reading from peer: %s\n", err.Error())

			if neterr, ok := err.(net.Error); ok {
				if neterr.Timeout() {
					dcEvent.Reason = "timeout"
				} else {
					dcEvent.Reason = "unrecoverable"
				}
			}

			peer.alive = false // this might not be necessary, we remove it in other places
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
