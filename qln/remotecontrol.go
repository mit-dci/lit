package qln

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/sig64"
)

func (nd *LitNode) RemoteControlRequestHandler(msg lnutil.RemoteControlRpcRequestMsg, peer *RemotePeer) error {

	log.Printf("Received remote control request from [%x]\nSignature:[%x]\n\n%s", msg.PubKey, msg.Sig, string(msg.Json))

	auth, err := nd.GetRemoteControlAuthorization(msg.PubKey)
	if err != nil {
		log.Printf("Error while checking authorization for remote control: %s", err.Error())
		return err
	}

	if !auth.Allowed {
		err = fmt.Errorf("Received remote control request from unauthorized peer: %x", msg.PubKey)
		log.Println(err.Error())
		return err
	}
	var digest []byte
	if msg.DigestType == lnutil.DIGEST_TYPE_SHA256 {
		hash := fastsha256.Sum256(msg.Json)
		digest = make([]byte, len(hash))
		copy(digest[:], hash[:])
	} else if msg.DigestType == lnutil.DIGEST_TYPE_RIPEMD160 {
		hash := btcutil.Hash160(msg.Json)
		digest = make([]byte, len(hash))
		copy(digest[:], hash[:])
	}

	pub, err := btcec.ParsePubKey(msg.PubKey[:], btcec.S256())
	if err != nil {
		log.Printf("Error parsing public key for remote control: %s", err.Error())
		return err
	}
	sig := sig64.SigDecompress(msg.Sig)
	signature, err := btcec.ParseDERSignature(sig, btcec.S256())
	if err != nil {
		log.Printf("Error parsing signature for remote control: %s", err.Error())
		return err
	}

	if !signature.Verify(digest, pub) {
		err = fmt.Errorf("Signature verification failed in remote control request")
		log.Println(err.Error())
		return err
	}

	obj := map[string]interface{}{}
	err = json.Unmarshal(msg.Json, &obj)
	if err != nil {
		log.Printf("Could not parse JSON: %s", err.Error())
		return err
	}

	go func() {
		var reply interface{}
		nd.LocalRPCCon.Call("LitRPC."+obj["method"].(string), obj["args"], reply)

		replyJSON, err := json.Marshal(reply)
		if err != nil {
			log.Printf("Could not produce reply JSON: %s", err.Error())
		}

		log.Printf("Reply for remote control: %s\n", replyJSON)
		/*
			response := lnutil.NewRemoteControlRpcResponseMsg(msg.Peer(), replyJSON)
			nd.OmniOut <- response*/
	}()
	return nil
}

func (nd *LitNode) RemoteControlResponseHandler(msg lnutil.RemoteControlRpcResponseMsg, peer *RemotePeer) error {
	log.Printf("Received remote control reply from peer %d:\n%s", msg.Peer(), string(msg.Json))
	return nil
}

// For now, this is simple: Allowed yes or no
// In the future allow more finegrained control
// over which RPCs are allowed and which are not,
// and perhaps authorize up to a certain amount for
// commands like send / push
type RemoteControlAuthorization struct {
	Allowed bool
}

func (r *RemoteControlAuthorization) Bytes() []byte {
	var buf bytes.Buffer

	if r.Allowed {
		buf.WriteByte(1)
	} else {
		buf.WriteByte(0)
	}
	return buf.Bytes()
}

func RemoteControlAuthorizationFromBytes(b []byte) *RemoteControlAuthorization {
	r := new(RemoteControlAuthorization)
	r.Allowed = false
	if b != nil && len(b) > 0 {
		r.Allowed = (b[0] == 1)
	}
	return r
}

func (nd *LitNode) SaveRemoteControlAuthorization(pub [33]byte, auth *RemoteControlAuthorization) error {
	if !auth.Allowed {
		return nd.RemoveRemoteControlAuthorization(pub)
	}
	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		// serialize state
		b := auth.Bytes()
		return cbk.Put(pub[:], b)
	})
}

func (nd *LitNode) RemoveRemoteControlAuthorization(pub [33]byte) error {
	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		return cbk.Delete(pub[:])
	})
}

func (nd *LitNode) GetRemoteControlAuthorization(pub [33]byte) (*RemoteControlAuthorization, error) {
	r := new(RemoteControlAuthorization)
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		// serialize state
		b := cbk.Get(pub[:])
		r = RemoteControlAuthorizationFromBytes(b)
		return nil
	})
	return r, err
}
