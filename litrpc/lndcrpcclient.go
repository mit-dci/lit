package litrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

type LndcRpcClient struct {
	lnconn           *lndc.Conn
	requestNonce     uint64
	requestNonceMtx  sync.Mutex
	responseChannels map[uint64]chan lnutil.RemoteControlRpcResponseMsg
	key              *btcec.PrivateKey
	StatusUpdates    chan lnutil.UIEventMsg
	conMtx           sync.Mutex
}

func LndcRpcCanConnectLocally() bool {
	litHomeDir := os.Getenv("HOME") + "/.lit"
	return LndcRpcCanConnectLocallyWithHomeDir(litHomeDir)
}

func LndcRpcCanConnectLocallyWithHomeDir(litHomeDir string) bool {
	keyFilePath := filepath.Join(litHomeDir, "privkey.hex")

	_, err := os.Stat(keyFilePath)
	return (err == nil)
}

func NewLocalLndcRpcClient() (*LndcRpcClient, error) {
	litHomeDir := os.Getenv("HOME") + "/.lit"
	return NewLocalLndcRpcClientWithHomeDirAndPort(litHomeDir, 2448)
}

func NewLocalLndcRpcClientWithPort(port uint32) (*LndcRpcClient, error) {
	litHomeDir := os.Getenv("HOME") + "/.lit"
	return NewLocalLndcRpcClientWithHomeDirAndPort(litHomeDir, port)
}

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
		log.Fatal(err)
	}
	var localIDPub [33]byte
	copy(localIDPub[:], localIDPriv.PubKey().SerializeCompressed())

	adr := fmt.Sprintf("%s@127.0.0.1:%d", lnutil.LitAdrFromPubkey(localIDPub), port)
	localIDPriv = nil

	return NewLndcRpcClient(adr, key)
}

func NewLndcRpcClient(address string, key *btcec.PrivateKey) (*LndcRpcClient, error) {
	var err error

	cli := new(LndcRpcClient)
	cli.responseChannels = make(map[uint64]chan lnutil.RemoteControlRpcResponseMsg)
	cli.StatusUpdates = make(chan lnutil.UIEventMsg)
	who, where := lnutil.ParseAdrString(address)

	// If we couldn't deduce a URL, look it up on the tracker
	if where == "" {
		err = fmt.Errorf("Tracker lookups not supported yet from LNDC proxy")
		if err != nil {
			return nil, err
		}
	}

	cli.lnconn, err = lndc.Dial(key, where, who, net.Dial)
	if err != nil {
		return nil, err
	}

	// Subscribe to status updates
	args := map[string]interface{}{"Subscribe": true}
	cli.Call("RemoteControl.SubscribeToUIEvents", args, nil)
	go cli.ReceiveLoop()
	return cli, nil
}

func (cli *LndcRpcClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	var err error

	cli.requestNonceMtx.Lock()
	cli.requestNonce++
	nonce := cli.requestNonce
	cli.requestNonceMtx.Unlock()

	cli.responseChannels[nonce] = make(chan lnutil.RemoteControlRpcResponseMsg)
	go func() {
		msg := new(lnutil.RemoteControlRpcRequestMsg)
		msg.Args, err = json.Marshal(args)
		msg.Idx = nonce
		msg.Method = serviceMethod

		if err != nil {
			panic(err)
		}

		rawMsg := msg.Bytes()
		cli.conMtx.Lock()
		n, err := cli.lnconn.Write(rawMsg)
		cli.conMtx.Unlock()
		if err != nil {
			panic(err)
		}

		if n < len(rawMsg) {
			panic(fmt.Errorf("Did not write entire message to peer"))
		}
	}()

	// If reply is nil the caller apparently doesn't care about the results. So we shouldn't wait for it
	if reply != nil {
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
			return errors.New("RPC call timed out")
		}
	}
	return nil
}

func (cli *LndcRpcClient) ReceiveLoop() {
	for {
		msg := make([]byte, 1<<24)
		//	log.Printf("read message from %x\n", l.RemoteLNId)
		n, err := cli.lnconn.Read(msg)
		if err != nil {
			log.Printf("Error reading message from LNDC: %s\n", err.Error())
			cli.lnconn.Close()
			return
		}
		msg = msg[:n]
		// We only care about RPC responses
		if msg[0] == lnutil.MSGID_REMOTE_RPCRESPONSE {
			response, err := lnutil.NewRemoteControlRpcResponseMsgFromBytes(msg, 0)
			if err != nil {
				log.Printf("Error while receiving RPC response: %s\n", err.Error())
				cli.lnconn.Close()
				return
			}

			responseChan, ok := cli.responseChannels[response.Idx]
			if ok {
				select {
				case responseChan <- response:
				default:
				}
				delete(cli.responseChannels, response.Idx)
			} else {
				log.Printf("Could not find response channel for index %d\n", response.Idx)
			}
		} else if msg[0] == lnutil.MSGID_UIEVENT_EVENT {
			response, err := lnutil.NewUIEventMsgFromBytes(msg, 0)
			if err != nil {
				log.Printf("Error receiving UI Event: %s", err.Error())
				cli.lnconn.Close()
				return
			}

			select {
			case cli.StatusUpdates <- response:
			default:
			}
		}
	}

}
