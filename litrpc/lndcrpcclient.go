package litrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/portxo"
)

// LndcRpcClient can be used to remotely talk to a lit node over LNDC, making
// remote control instructions over the remote control interface. It uses a
// regular lndc.Conn to connect to lit over port 2448, and sends
// RemoteControlRpcRequestMsg to it, and receives RemoteControlRpcResponseMsg
type LndcRpcClient struct {
	lnconn             *lndc.Conn
	requestNonce       uint64
	requestNonceMtx    sync.Mutex
	responseChannelMtx sync.Mutex
	responseChannels   map[uint64]chan *lnutil.RemoteControlRpcResponseMsg
	key                *koblitz.PrivateKey
	conMtx             sync.Mutex

	chunksOfMsg map[int64]*lnutil.ChunkMsg
}

// LndcRpcCanConnectLocally checks if we can connect to lit using the normal
// home directory. In that case, we read from the privkey.hex and use a different
// derivation than the nodeID to determine the private key. This key is authorized
// by default for remote control.
func LndcRpcCanConnectLocally() bool {
	litHomeDir := os.Getenv("HOME") + "/.lit"
	return LndcRpcCanConnectLocallyWithHomeDir(litHomeDir)
}

// LndcRpcCanConnectLocallyWithHomeDir checks if we can connect to lit given the
// home directory. In that case, we read from the privkey.hex and use a different
// derivation than the nodeID to determine the private key. This key is authorized
// by default for remote control.
func LndcRpcCanConnectLocallyWithHomeDir(litHomeDir string) bool {
	keyFilePath := filepath.Join(litHomeDir, "privkey.hex")

	_, err := os.Stat(keyFilePath)
	return (err == nil)
}

// NewLocalLndcRpcClient is an overload for NewLocalLndcRpcClientWithHomeDirAndPort
// using the default home dir and port
func NewLocalLndcRpcClient() (*LndcRpcClient, error) {
	litHomeDir := os.Getenv("HOME") + "/.lit"
	return NewLocalLndcRpcClientWithHomeDirAndPort(litHomeDir, 2448)
}

// NewLocalLndcRpcClientWithPort is an overload for
// NewLocalLndcRpcClientWithHomeDirAndPort using the default home dir
func NewLocalLndcRpcClientWithPort(port uint32) (*LndcRpcClient, error) {
	litHomeDir := os.Getenv("HOME") + "/.lit"
	return NewLocalLndcRpcClientWithHomeDirAndPort(litHomeDir, port)
}

// NewLocalLndcRpcClientWithHomeDirAndPort loads up privkey.hex, and derives
// the local lit node's address from it, as well as derives the default remote
// control private key from it. Then it will connect to the local lit instance.
func NewLocalLndcRpcClientWithHomeDirAndPort(litHomeDir string, port uint32) (*LndcRpcClient, error) {
	keyFilePath := filepath.Join(litHomeDir, "privkey.hex")
	privKey, err := lnutil.ReadKeyFile(keyFilePath)
	if err != nil {
		return nil, err
	}
	rootPrivKey, err := hdkeychain.NewMaster(privKey[:], &coinparam.TestNet3Params)
	if err != nil {
		return nil, err
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = 513 | 1<<31
	kg.Step[2] = 9 | 1<<31
	kg.Step[3] = 1 | 1<<31
	kg.Step[4] = 0 | 1<<31
	key, err := kg.DerivePrivateKey(rootPrivKey)
	if err != nil {
		return nil, err
	}

	kg.Step[3] = 0 | 1<<31
	localIDPriv, err := kg.DerivePrivateKey(rootPrivKey)
	if err != nil {
		logging.Errorf(err.Error())
	}
	var localIDPub [33]byte
	copy(localIDPub[:], localIDPriv.PubKey().SerializeCompressed())

	adr := fmt.Sprintf("%s@127.0.0.1:%d", lnutil.LitAdrFromPubkey(localIDPub), port)
	localIDPriv = nil

	return NewLndcRpcClient(adr, key)
}

// NewLndcRpcClient creates a new LNDC client using the given private key, which
// is arbitrary. It will then connect to the lit node specified in address, and
// can then exchange remote control calls with it. In order to succesfully
// execute command, the given key must be authorized in the lit instance we're
// connecting to.
func NewLndcRpcClient(address string, key *koblitz.PrivateKey) (*LndcRpcClient, error) {
	var err error

	cli := new(LndcRpcClient)

	cli.chunksOfMsg = make(map[int64]*lnutil.ChunkMsg)

	// Create a map of chan objects to receive returned responses on. These channels
	// are sent to from the ReceiveLoop, and awaited in the Call method.
	cli.responseChannels = make(map[uint64]chan *lnutil.RemoteControlRpcResponseMsg)

	//Parse the address we're connecting to
	who, where := lnutil.ParseAdrString(address)

	// If we couldn't deduce a URL, look it up on the tracker
	if where == "" {
		// TODO: Implement address lookups
		err = fmt.Errorf("Tracker lookups not supported yet from LNDC proxy")
		if err != nil {
			return nil, err
		}
	}

	// Dial a connection to the lit node
	cli.lnconn, err = lndc.Dial(key, where, who, net.Dial)
	if err != nil {
		return nil, err
	}

	// Start the receive loop for reply messages
	go cli.ReceiveLoop()
	return cli, nil
}

func (cli *LndcRpcClient) Close() error {
	return cli.lnconn.Close()
}

func (cli *LndcRpcClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	var err error

	// Generate a local unique nonce using the mutex
	cli.requestNonceMtx.Lock()
	cli.requestNonce++
	nonce := cli.requestNonce
	cli.requestNonceMtx.Unlock()

	// Create the channel to receive the reply on
	cli.responseChannelMtx.Lock()
	cli.responseChannels[nonce] = make(chan *lnutil.RemoteControlRpcResponseMsg)
	cli.responseChannelMtx.Unlock()

	// Send the message in a goroutine
	go func() {
		msg := new(lnutil.RemoteControlRpcRequestMsg)
		msg.Args, err = json.Marshal(args)
		msg.Idx = nonce
		msg.Method = serviceMethod

		if err != nil {
			logging.Fatal(err)
		}

		rawMsg := msg.Bytes()
		cli.conMtx.Lock()
		n, err := cli.lnconn.Write(rawMsg)
		cli.conMtx.Unlock()
		if err != nil {
			logging.Fatal(err)
		}

		if n < len(rawMsg) {
			logging.Fatal(fmt.Errorf("Did not write entire message to peer"))
		}
	}()

	// If reply is nil the caller apparently doesn't care about the results. So we shouldn't wait for it
	if reply != nil {
		// If not nil, await the reply from the responseChannel for the nonce we sent out.
		// the server will include the same nonce in its reply.
		select {
		case receivedReply := <-cli.responseChannels[nonce]:
			{
				if receivedReply.Error {
					return errors.New(string(receivedReply.Result))
				}

				err = json.Unmarshal(receivedReply.Result, &reply)
				return err
			}
		case <-time.After(time.Second * 10):
			// If no reply is received within 10 seconds, we time out the request.
			// TODO: We could make this configurable in the call
			return errors.New("RPC call timed out")
		}
	}
	return nil
}

// ReceiveLoop reads messages from the LNDC connection and check if they are
// RPC responses
func (cli *LndcRpcClient) ReceiveLoop() {
	for {
		msg := make([]byte, 1<<24)
		//	log.Printf("read message from %x\n", l.RemoteLNId)
		n, err := cli.lnconn.Read(msg)
		if err != nil {
			logging.Warnf("Error reading message from LNDC: %s\n", err.Error())
			cli.lnconn.Close()
			return
		}
		msg = msg[:n]

		if msg[0] == lnutil.MSGID_CHUNKS_BEGIN {

			beginChunksMsg, _ := lnutil.NewChunksBeginMsgFromBytes(msg, 0)

			msg_tmp := new(lnutil.ChunkMsg)
			msg_tmp.TimeStamp = beginChunksMsg.TimeStamp
			cli.chunksOfMsg[beginChunksMsg.TimeStamp] = msg_tmp
			
			continue
		}

		if msg[0] == lnutil.MSGID_CHUNK_BODY {

			chunkMsg, _ := lnutil.NewChunkMsgFromBytes(msg, 0)
			cli.chunksOfMsg[chunkMsg.TimeStamp].Data = append(cli.chunksOfMsg[chunkMsg.TimeStamp].Data, chunkMsg.Data...)

			continue
		}
		
		if msg[0] == lnutil.MSGID_CHUNKS_END {

			endChunksMsg, _ := lnutil.NewChunksBeginMsgFromBytes(msg, 0)
			msg = cli.chunksOfMsg[endChunksMsg.TimeStamp].Data

		}		



		// We only care about RPC responses (for now)
		if msg[0] == lnutil.MSGID_REMOTE_RPCRESPONSE {
			// Parse the received message
			response, err := lnutil.NewRemoteControlRpcResponseMsgFromBytes(msg, 0)
			if err != nil {
				logging.Warnf("Error while receiving RPC response: %s\n", err.Error())
				cli.lnconn.Close()
				return
			}

			// Find the response channel to send the reply to
			responseChan, ok := cli.responseChannels[response.Idx]
			if ok {
				// Send the response, but don't depend on someone
				// listening. The caller decides if he's interested in the
				// reply and therefore, it could have not blocked and just
				// ignore the return value.
				select {
				case responseChan <- &response:
				default:
				}

				// Clean up the channel to preserve memory. It's only used once.
				cli.responseChannelMtx.Lock()
				delete(cli.responseChannels, response.Idx)
				cli.responseChannelMtx.Unlock()

			} else {
				logging.Errorf("Could not find response channel for index %d\n", response.Idx)
			}
		}
	}

}
