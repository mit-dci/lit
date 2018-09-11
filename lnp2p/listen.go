package lnp2p

import (
	"github.com/mit-dci/lit/eventbus"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnio"
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
			// TODO logging
			break // usually means the socket was closed somehow
		}

		lndcConn, ok := netConn.(*lndc.Conn)
		if !ok {
			// TODO logging
			// this should never happen
			continue
		}

		// TODO logging

		rpk := pubkey(lndcConn.RemotePub())
		rlitaddr := convertPubkeyToLitAddr(rpk)
		rnetaddr := lndcConn.RemoteAddr()

		// Read the peer info from the DB.
		pi, err := pm.peerdb.GetPeerInfo(rlitaddr)
		if err != nil {
			netConn.Close()
			// TODO logging
			// TODO handle this better
			continue
		}

		// Add the peer data to the DB if we don't have it.
		if pi == nil {
			pi = &lnio.PeerInfo{
				Nickname: nil,
				NetAddr:  rnetaddr.String(),
			}
			err = pm.peerdb.AddPeer(rlitaddr, *pi)
			if err != nil {
				// don't close it, I guess
				// TODO logging
				// TODO handle this better
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

		// Add it to the peer manager.
		pm.mtx.Lock()
		pm.registerPeer(npeer)
		pm.mtx.Unlock()

		// TODO Start read thread.
		go processConnectionFeed(npeer, pm)

	}

	// Update the stop reason.
	stopEvent.Reason = "closed"

	// Then delete the entry from listening ports.
	pm.mtx.Lock()
	delete(pm.listeningPorts, listenAddr)
	pm.mtx.Unlock()

	// after this the stop event will be published

}

func processConnectionFeed(peer *Peer, pm *PeerManager) {

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
			// TODO Error handling.
			// TODO Logging.
			peer.conn.Close()
			return
		}

		// Truncate it.
		buf = buf[:n]

		// Parse it into an actual message.
		m, err := processMessage(buf, peer)
		if err != nil {
			// TODO ERror handling.
			// TODO Logging.
			peer.conn.Close()
			return
		}

		// Publish the event to the event bus so qln or something can pick it up.
		// TODO Refactor this.
		pm.ebus.Publish(NetMessageRecvEvent{&m, peer, buf})

	}

}

func publishStopEvent(event *StopListeningPortEvent, ebus *eventbus.EventBus) {
	ebus.Publish(*event)
}

func publishDisconnectEvent(event *PeerDisconnectEvent, ebus *eventbus.EventBus) {
	ebus.Publish(*event)
}
