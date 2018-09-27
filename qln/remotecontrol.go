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
	"github.com/mit-dci/lit/crypto/fastsha256"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/sig64"
)

// RCRequestAuthArgs contains the required parameters
// to request authorization for remote control. Only
// the pub key is necessary
type RCRequestAuthArgs struct {
	// Public Key to authorize
	PubKey [33]byte
}

// RemoteControlRequestHandler handles an incoming remote control request
func (nd *LitNode) RemoteControlRequestHandler(msg lnutil.RemoteControlRpcRequestMsg, peer *RemotePeer) error {
	var pubKey [33]byte
	// transportAuthenticated will store whether the request authorization
	// should be checked against the transport's public key, or the one
	// that's included in the message.
	transportAuthenticated := true
	copy(pubKey[:], peer.Con.RemotePub().SerializeCompressed())

	if msg.PubKey != [33]byte{} {
		// The message contains a pubkey. So use that to authorize. We also
		// need to verify the signature inside the message then. If the
		// transport authorization is used, we can skip that because the
		// transport is already secured using signatures.
		pubKey = msg.PubKey
		transportAuthenticated = false
	}
	logging.Infof("Received remote control request [%s] from [%x]\n\n%s", msg.Method, pubKey, string(msg.Args))

	// Fetch the remote control authorization based on the used public key
	auth, err := nd.GetRemoteControlAuthorization(pubKey)
	if err != nil {
		logging.Errorf("Error while checking authorization for remote control: %s", err.Error())
		return err
	}

	// Whitelisted method(s) - Methods that don't require authorization.
	whitelisted := false
	if msg.Method == "LitRPC.RequestRemoteControlAuthorization" {
		// Request for authorization. You do not need to be authorized for this
		// All this method does is include the remote caller's public key into
		// the remote control authorization database as "requested authorization"
		// It will remain unauthorized until approved.
		whitelisted = true
		args := new(RCRequestAuthArgs)
		args.PubKey = pubKey
		msg.Args, err = json.Marshal(args)
		if err != nil {
			logging.Errorf("Error while updating RequestRemoteControlAuthorization arguments: %s", err.Error())
			return err
		}
	}

	// Method to check if the remote caller is authorized. This is not actually
	// part of the RPC surface, but only for the remote control. Hence it being
	// in a different namespace and returning a result from this method directly
	// without calling RPC methods.
	if msg.Method == "RemoteControl.CheckAuthorizationStatus" {
		resp, err := json.Marshal(map[string]interface{}{"Authorized": auth.Allowed})
		if err != nil {
			return err
		}
		outMsg := lnutil.NewRemoteControlRpcResponseMsg(peer.Idx, msg.Idx, false, resp)
		nd.tmpSendLitMsg(outMsg)
		return nil
	}

	// If i'm not authorized, and it's not a whitelisted method then we fail the
	// request with an 'unauthorized' error
	if !auth.Allowed && !whitelisted {
		err = fmt.Errorf("Received remote control request from unauthorized peer: %x", pubKey)
		logging.Errorf(err.Error())

		outMsg := lnutil.NewRemoteControlRpcResponseMsg(peer.Idx, msg.Idx, true, []byte("Unauthorized"))
		nd.tmpSendLitMsg(outMsg)

		return err
	}

	// If the transport is not authenticated, and we're using the key provided
	// in the message, then a signature is required to verify the caller actually
	// controls that given key.
	if !transportAuthenticated {
		msgSig := msg.Sig

		// Make the signature empty. The caller signed the message byte slice
		// with an empty signature.
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

		pub, err := koblitz.ParsePubKey(msg.PubKey[:], koblitz.S256())
		if err != nil {
			logging.Errorf("Error parsing public key for remote control: %s", err.Error())
			return err
		}
		sig := sig64.SigDecompress(msgSig)
		signature, err := koblitz.ParseDERSignature(sig, koblitz.S256())
		if err != nil {
			logging.Errorf("Error parsing signature for remote control: %s", err.Error())
			return err
		}

		if !signature.Verify(digest, pub) {
			err = fmt.Errorf("Signature verification failed in remote control request")
			logging.Errorf(err.Error())
			return err
		}
	}

	// Check if the arguments property is valid JSON by deserializing it
	obj := map[string]interface{}{}
	err = json.Unmarshal(msg.Args, &obj)
	if err != nil {
		logging.Errorf("Could not parse JSON: %s", err.Error())
		return err
	}

	// Handle the request in a goroutine
	go func() {
		// Sanity check for the method
		if !strings.HasPrefix(msg.Method, "LitRPC.") {
			logging.Warnf("Remote control method does not start with `LitRPC.`. We don't know any better. Yet.")
			return
		}

		// Use reflection to call the method on the RPC object
		methodName := strings.TrimPrefix(msg.Method, "LitRPC.")
		rpcType := reflect.ValueOf(nd.RPC)
		if rpcType.IsValid() {
			method := rpcType.MethodByName(methodName)
			if method.IsValid() {

				// Our RPC calls always have params as (args, reply)
				argsType := method.Type().In(0)
				argsPointer := false

				// Unfortunately, sometimes the arguments to the function need
				// a pointer, and sometimes a value. So we check and adjust.
				if argsType.Kind() == reflect.Ptr {
					argsPointer = true
					argsType = argsType.Elem()
				}

				argsPayload := reflect.New(argsType)

				// Unmarshal the JSON into the args object
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

				outMsg := lnutil.NewRemoteControlRpcResponseMsg(peer.Idx, msg.Idx, replyIsError, reply)
				nd.tmpSendLitMsg(outMsg)
			}
		}
	}()
	return nil
}

// RemoteControlResponseHandler handles the remote control response messages.
// At this time, this is not used in lit itself. Remote control messages are
// sent from the LndcRpcClient and responses are handled there. In normal operation
// two regular lit nodes do not talk to each other using remote control. But
// just in case someone sends us one, we print it out here.
func (nd *LitNode) RemoteControlResponseHandler(msg lnutil.RemoteControlRpcResponseMsg, peer *RemotePeer) error {
	logging.Debugf("Received remote control reply from peer %d:\n%s", peer.Idx, string(msg.Result))
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

// Bytes serializes a remotecontrol authorization into a byteslice
func (r *RemoteControlAuthorization) Bytes() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.BigEndian, r.Allowed)
	binary.Write(&buf, binary.BigEndian, r.UnansweredRequest)
	return buf.Bytes()
}

// RemoteControlAuthorizationFromBytes parses a byteslice into a
// RemoteControlAuthorization object
func RemoteControlAuthorizationFromBytes(b []byte, pubKey [33]byte) *RemoteControlAuthorization {
	r := new(RemoteControlAuthorization)
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &r.Allowed)
	binary.Read(buf, binary.BigEndian, &r.UnansweredRequest)
	r.PubKey = pubKey
	return r
}

// SaveRemoteControlAuthorization saves the authorization for a specific
// pubkey into the database.
func (nd *LitNode) SaveRemoteControlAuthorization(pub [33]byte, auth *RemoteControlAuthorization) error {
	return nd.LitDB.Update(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		// serialize state
		b := auth.Bytes()
		return cbk.Put(pub[:], b)
	})
}

// GetRemoteControlAuthorization retrieves the remote controlauthorizzation for
// a specific pubkey from the database.
func (nd *LitNode) GetRemoteControlAuthorization(pub [33]byte) (*RemoteControlAuthorization, error) {
	r := new(RemoteControlAuthorization)

	// If the client uses our default remote control key (derived from our root priv)
	// then it has access to our private key (file) and is most likely running from our
	// localhost. So we always accept this.
	if bytes.Equal(pub[:], nd.DefaultRemoteControlKey.SerializeCompressed()) {
		r.Allowed = true
		return r, nil
	}

	// Fetch the authorization from the database and return it.
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		// serialize state
		b := cbk.Get(pub[:])
		r = RemoteControlAuthorizationFromBytes(b, pub)
		return nil
	})
	return r, err
}

// GetPendingRemoteControlRequests retrieves all pending remote control
// authorization requests, so that a GUI can print them out for the user to
// authorize or not.
func (nd *LitNode) GetPendingRemoteControlRequests() ([]*RemoteControlAuthorization, error) {
	r := make([]*RemoteControlAuthorization, 0)
	err := nd.LitDB.View(func(btx *bolt.Tx) error {
		cbk := btx.Bucket(BKTRCAuth)
		// serialize state
		err := cbk.ForEach(func(k, v []byte) error {
			logging.Debugf("%x : %s\n", k, v)
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
