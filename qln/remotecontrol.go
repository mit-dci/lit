package qln

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/sig64"
)

func (nd *LitNode) RemoteControlRequestHandler(msg lnutil.RemoteControlRpcRequestMsg, peer *RemotePeer) error {
	var pubKey [33]byte
	transportAuthenticated := true
	copy(pubKey[:], peer.Con.RemotePub().SerializeCompressed())

	if msg.PubKey != [33]byte{} {
		pubKey = msg.PubKey
		transportAuthenticated = false
	}
	log.Printf("Received remote control request [%s] from [%x]\n\n%s", msg.Method, pubKey, string(msg.Args))

	auth, err := nd.GetRemoteControlAuthorization(pubKey)
	if err != nil {
		log.Printf("Error while checking authorization for remote control: %s", err.Error())
		return err
	}

	if !auth.Allowed {
		err = fmt.Errorf("Received remote control request from unauthorized peer: %x", pubKey)
		log.Println(err.Error())

		outMsg := lnutil.NewRemoteControlRpcResponseMsg(msg.Peer(), msg.Idx, true, []byte("Unauthorized"))
		nd.OmniOut <- outMsg

		return err

	}

	if !transportAuthenticated {
		// If the message specifies a pubkey, then we haven't authenticated this message via the
		// lndc transport already. So we need to check the signature embedded in the message.

		var hashBuf bytes.Buffer
		hashBuf.Write(msg.Args)
		hashBuf.Write([]byte(msg.Method))
		binary.Write(&hashBuf, binary.BigEndian, msg.Idx)

		var digest []byte
		if msg.DigestType == lnutil.DIGEST_TYPE_SHA256 {
			hash := fastsha256.Sum256(hashBuf.Bytes())
			digest = make([]byte, len(hash))
			copy(digest[:], hash[:])
		} else if msg.DigestType == lnutil.DIGEST_TYPE_RIPEMD160 {
			hash := btcutil.Hash160(hashBuf.Bytes())
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
	}

	obj := map[string]interface{}{}
	err = json.Unmarshal(msg.Args, &obj)
	if err != nil {
		log.Printf("Could not parse JSON: %s", err.Error())
		return err
	}

	go func() {
		if !strings.HasPrefix(msg.Method, "LitRPC.") {
			log.Printf("Remote control method does not start with `LitRPC.`. We don't know any better. Yet.")
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
					log.Printf("Error parsing json argument: %s", err.Error())
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
	log.Printf("Received remote control reply from peer %d:\n%s", msg.Peer(), string(msg.Result))
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

	// If the client uses our default remote control key (derived from our root priv)
	// then it has access to our private key (file) and is most likely running from our
	// localhost. So we always accept this.
	if bytes.Equal(pub[:], nd.DefaultRemoteControlKey.SerializeCompressed()) {
		log.Printf("Received key [%x] matches default key [%x], so allowing\n", pub[:], nd.DefaultRemoteControlKey.SerializeCompressed())

		r.Allowed = true
		return r, nil
	}
	log.Printf("Received key [%x] doesnt default key [%x], so allowing\n", pub[:], nd.DefaultRemoteControlKey.SerializeCompressed())

	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		// serialize state
		b := cbk.Get(pub[:])
		r = RemoteControlAuthorizationFromBytes(b)
		return nil
	})
	return r, err
}
