package qln

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/sig64"
)

type RCRequestAuthArgs struct {
	PubKey [33]byte
}

func (nd *LitNode) RemoteControlRequestHandler(msg lnutil.RemoteControlRpcRequestMsg, peer *RemotePeer) error {
	var pubKey [33]byte
	transportAuthenticated := true
	copy(pubKey[:], peer.Con.RemotePub().SerializeCompressed())

	if msg.PubKey != [33]byte{} {
		pubKey = msg.PubKey
		transportAuthenticated = false
	}
	logging.Infof("Received remote control request [%s] from [%x]\n\n%s", msg.Method, pubKey, string(msg.Args))

	auth, err := nd.GetRemoteControlAuthorization(pubKey)
	if err != nil {
		logging.Errorf("Error while checking authorization for remote control: %s", err.Error())
		return err
	}

	// Whitelisted method(s)
	whitelisted := false
	if msg.Method == "LitRPC.RequestRemoteControlAuthorization" {
		whitelisted = true
		args := new(RCRequestAuthArgs)
		args.PubKey = pubKey
		msg.Args, err = json.Marshal(args)
		if err != nil {
			logging.Errorf("Error while updating RequestRemoteControlAuthorization arguments: %s", err.Error())
			return err
		}
	}

	if msg.Method == "RemoteControl.CheckAuthorizationStatus" {
		resp, err := json.Marshal(map[string]interface{}{"Authorized": auth.Allowed})
		if err != nil {
			return err
		}
		outMsg := lnutil.NewRemoteControlRpcResponseMsg(msg.Peer(), msg.Idx, false, resp)
		nd.OmniOut <- outMsg
		return nil
	}

	if !auth.Allowed && !whitelisted {
		err = fmt.Errorf("Received remote control request from unauthorized peer: %x", pubKey)
		logging.Errorln(err.Error())

		outMsg := lnutil.NewRemoteControlRpcResponseMsg(msg.Peer(), msg.Idx, true, []byte("Unauthorized"))
		nd.OmniOut <- outMsg

		return err
	}

	if !transportAuthenticated {
		// If the message specifies a pubkey, then we haven't authenticated this message via the
		// lndc transport already. So we need to check the signature embedded in the message.
		msgSig := msg.Sig
		msg.Sig = [64]byte{}

		var digest []byte
		if msg.DigestType == lnutil.DIGEST_TYPE_SHA256 {
			hash := fastsha256.Sum256(msg.Bytes())
			digest = make([]byte, len(hash))
			copy(digest[:], hash[:])
		} else if msg.DigestType == lnutil.DIGEST_TYPE_RIPEMD160 {
			hash := btcutil.Hash160(msg.Bytes())
			digest = make([]byte, len(hash))
			copy(digest[:], hash[:])
		}

		pub, err := btcec.ParsePubKey(msg.PubKey[:], btcec.S256())
		if err != nil {
			logging.Errorf("Error parsing public key for remote control: %s", err.Error())
			return err
		}
		sig := sig64.SigDecompress(msgSig)
		signature, err := btcec.ParseDERSignature(sig, btcec.S256())
		if err != nil {
			logging.Errorf("Error parsing signature for remote control: %s", err.Error())
			return err
		}

		if !signature.Verify(digest, pub) {
			err = fmt.Errorf("Signature verification failed in remote control request")
			logging.Errorln(err.Error())
			return err
		}
	}

	obj := map[string]interface{}{}
	err = json.Unmarshal(msg.Args, &obj)
	if err != nil {
		logging.Errorf("Could not parse JSON: %s", err.Error())
		return err
	}
	go func() {
		if !strings.HasPrefix(msg.Method, "LitRPC.") {
			logging.Warnf("Remote control method does not start with `LitRPC.`. We don't know any better. Yet.")
			return
		}
		methodName := strings.TrimPrefix(msg.Method, "LitRPC.")
		rpcType := reflect.ValueOf(nd.RPC)
		if rpcType.IsValid() {
			method := rpcType.MethodByName(methodName)
			if method.IsValid() {

				// Our RPC calls always have params as (args, reply)
				argsType := method.Type().In(0)
				argsPointer := false
				if argsType.Kind() == reflect.Ptr {
					argsPointer = true
					argsType = argsType.Elem()
				}

				argsPayload := reflect.New(argsType)

				err = json.Unmarshal(msg.Args, argsPayload.Interface())
				if err != nil {
					logging.Errorf("Error parsing json argument: %s", err.Error())
					return
				}

				replyType := method.Type().In(1).Elem()
				replyPayload := reflect.New(replyType)

				if !argsPointer {
					argsPayload = argsPayload.Elem()
				}
				result := method.Call([]reflect.Value{argsPayload, replyPayload})

				var reply []byte
				replyIsError := false
				if !result[0].IsNil() {
					replyIsError = true
					err = result[0].Interface().(error)
					reply = []byte(err.Error())
				} else {
					reply, err = json.Marshal(replyPayload.Interface())
					if err != nil {
						replyIsError = true
						reply = []byte(err.Error())
					}
				}

				outMsg := lnutil.NewRemoteControlRpcResponseMsg(msg.Peer(), msg.Idx, replyIsError, reply)
				nd.OmniOut <- outMsg
			}
		}
	}()
	return nil
}

func (nd *LitNode) RemoteControlResponseHandler(msg lnutil.RemoteControlRpcResponseMsg, peer *RemotePeer) error {
	logging.Infof("Received remote control reply from peer %d:\n%s", msg.Peer(), string(msg.Result))
	return nil
}

// For now, this is simple: Allowed yes or no
// In the future allow more finegrained control
// over which RPCs are allowed and which are not,
// and perhaps authorize up to a certain amount for
// commands like send / push
type RemoteControlAuthorization struct {
	PubKey            [33]byte
	Allowed           bool
	UnansweredRequest bool
}

func (r *RemoteControlAuthorization) Bytes() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, r.Allowed)
	binary.Write(&buf, binary.BigEndian, r.UnansweredRequest)
	return buf.Bytes()
}

func RemoteControlAuthorizationFromBytes(b []byte, pubKey [33]byte) *RemoteControlAuthorization {
	r := new(RemoteControlAuthorization)
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &r.Allowed)
	binary.Read(buf, binary.BigEndian, &r.UnansweredRequest)
	r.PubKey = pubKey
	return r
}

func (nd *LitNode) SaveRemoteControlAuthorization(pub [33]byte, auth *RemoteControlAuthorization) error {
	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		// serialize state
		b := auth.Bytes()
		return cbk.Put(pub[:], b)
	})
}

func (nd *LitNode) GetRemoteControlAuthorization(pub [33]byte) (*RemoteControlAuthorization, error) {
	r := new(RemoteControlAuthorization)

	// If the client uses our default remote control key (derived from our root priv)
	// then it has access to our private key (file) and is most likely running from our
	// localhost. So we always accept this.
	if bytes.Equal(pub[:], nd.DefaultRemoteControlKey.SerializeCompressed()) {
		r.Allowed = true
		return r, nil
	}

	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		// serialize state
		b := cbk.Get(pub[:])
		r = RemoteControlAuthorizationFromBytes(b, pub)
		return nil
	})
	return r, err
}

func (nd *LitNode) GetPendingRemoteControlRequests() ([]*RemoteControlAuthorization, error) {
	r := make([]*RemoteControlAuthorization, 0)
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		// serialize state
		err := cbk.ForEach(func(k, v []byte) error {
			logging.Infof("%x : %s\n", k, v)
			if len(v) >= 2 {
				if v[1] != 0x00 {
					var pubKey [33]byte
					copy(pubKey[:], k)
					r = append(r, RemoteControlAuthorizationFromBytes(v, pubKey))
				}
			}
			return nil
		})
		return err
	})
	return r, err
}
