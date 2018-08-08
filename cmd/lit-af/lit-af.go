package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/btcutil/hdkeychain"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/portxo"
)

/*
Lit-AF

The Lit Advanced Functionality interface.
This is a text mode interface to lit.  It connects over jsonrpc to the a lit
node and tells that lit node what to do.  The lit node also responds so that
lit-af can tell what's going on.

lit-gtk does most of the same things with a gtk interface, but there will be
some yet-undefined advanced functionality only available in lit-af.

May end up using termbox-go

*/

//// BalReply is the reply when the user asks about their balance.
//// This is a Non-Channel
//type BalReply struct {
//	ChanTotal         int64
//	TxoTotal          int64
//	SpendableNow      int64
//	SpendableNowWitty int64
//}

const (
	litHomeDirName     = ".lit"
	historyFilename    = "lit-af.history"
	defaultKeyFileName = "privkey.hex"
)

type litAfClient struct {
	remote           string
	port             uint16
	lnconn           *lndc.Conn
	addr             string
	litHomeDir       string
	requestNonce     uint64
	requestNonceMtx  sync.Mutex
	responseChannels map[uint64]chan lnutil.RemoteControlRpcResponseMsg
	key              *btcec.PrivateKey
}

type Command struct {
	Format           string
	Description      string
	ShortDescription string
}

func setConfig(lc *litAfClient) {
	hostptr := flag.String("node", "127.0.0.1", "host to connect to")
	portptr := flag.Int("p", 2448, "port to connect to")
	dirptr := flag.String("dir", filepath.Join(os.Getenv("HOME"), litHomeDirName), "directory to save settings")
	addrptr := flag.String("addr", "", "address of the host we're connecting to (if not running on the same machine as lit)")

	flag.Parse()

	lc.remote = *hostptr
	lc.port = uint16(*portptr)
	lc.litHomeDir = *dirptr
	lc.addr = *addrptr

	if lnutil.LitAdrOK(lc.addr) && lc.remote == "127.0.0.1" {
		// probably need to look up when addr is set but node is still localhost
	}
}

// for now just testing how to connect and get messages back and forth
func main() {
	lc := new(litAfClient)
	setConfig(lc)

	keyFilePath := filepath.Join(lc.litHomeDir, defaultKeyFileName)
	remote := false

	// create home directory if it does not exist
	_, err := os.Stat(lc.litHomeDir)
	if os.IsNotExist(err) {
		os.Mkdir(lc.litHomeDir, 0700)
	}

	if _, err := os.Stat(keyFilePath); os.IsNotExist(err) {
		// If the keyfile does not exist, we're probably not running on the
		// same machine as lit. So we have a remote connection. Which is fine,
		// but then we need a little more info.

		// Plus, just to be sure we'll save the local key in a different file
		keyFilePath = filepath.Join(lc.litHomeDir, "lit-af-key.hex")
		remote = true

		if !lnutil.LitAdrOK(lc.addr) {
			log.Fatal("Since you are remotely connecting to lit, you need to specify the node's LN address using the -addr parameter")
		}
	}

	// read key file (generate if not found)
	privKey, err := lnutil.ReadKeyFile(keyFilePath)
	if err != nil {
		log.Fatal(err)
	}
	rootPrivKey, err := hdkeychain.NewMaster(privKey[:], &coinparam.TestNet3Params)
	if err != nil {
		log.Fatal(err)
	}

	var kg portxo.KeyGen
	kg.Depth = 5
	kg.Step[0] = 44 | 1<<31
	kg.Step[1] = 513 | 1<<31
	kg.Step[2] = 9 | 1<<31
	kg.Step[3] = 1 | 1<<31
	kg.Step[4] = 0 | 1<<31
	lc.key, err = kg.DerivePrivateKey(rootPrivKey)
	if err != nil {
		log.Fatal(err)
	}

	localLNAdr := lc.addr

	if !remote {
		kg.Step[3] = 0 | 1<<31
		localIDPriv, err := kg.DerivePrivateKey(rootPrivKey)
		if err != nil {
			log.Fatal(err)
		}
		var localIDPub [33]byte
		copy(localIDPub[:], localIDPriv.PubKey().SerializeCompressed())
		localLNAdr = lnutil.LitAdrFromPubkey(localIDPub)
		localIDPriv = nil
	}

	privKey = nil
	rootPrivKey = nil

	lc.responseChannels = make(map[uint64]chan lnutil.RemoteControlRpcResponseMsg)

	addr := fmt.Sprintf("%s:%d", lc.remote, lc.port)
	lc.lnconn, err = lndc.Dial(lc.key, addr, localLNAdr, net.Dial)
	if err != nil {
		log.Fatal(err)
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       lnutil.Prompt("lit-af") + lnutil.White("# "),
		HistoryFile:  filepath.Join(lc.litHomeDir, historyFilename),
		AutoComplete: lc.NewAutoCompleter(),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()
	go lc.ReceiveLoop()

	// main shell loop
	for {
		// setup reader with max 4K input chars
		msg, err := rl.Readline()
		if err != nil {
			break
		}
		msg = strings.TrimSpace(msg)
		if len(msg) == 0 {
			continue
		}
		rl.SaveHistory(msg)

		cmdslice := strings.Fields(msg)                         // chop input up on whitespace
		fmt.Fprintf(color.Output, "entered command: %s\n", msg) // immediate feedback

		err = lc.Shellparse(cmdslice)
		if err != nil { // only error should be user exit
			log.Fatal(err)
		}
	}

	//	err = c.Call("LitRPC.Bal", nil, &br)
	//	if err != nil {
	//		log.Fatal("rpc call error:", err)
	//	}
	//	fmt.Printf("Sent bal req, response: txototal %d\n", br.TxoTotal)
}

func (lc *litAfClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	var err error
	lc.requestNonceMtx.Lock()
	lc.requestNonce++
	nonce := lc.requestNonce
	lc.requestNonceMtx.Unlock()

	lc.responseChannels[nonce] = make(chan lnutil.RemoteControlRpcResponseMsg)
	go func() {
		msg := new(lnutil.RemoteControlRpcRequestMsg)
		msg.Args, err = json.Marshal(args)
		msg.Idx = nonce
		msg.Method = serviceMethod

		if err != nil {
			panic(err)
		}

		rawMsg := msg.Bytes()
		n, err := lc.lnconn.Write(rawMsg)
		if err != nil {
			panic(err)
		}

		if n < len(rawMsg) {
			panic(fmt.Errorf("Did not write entire message to peer"))
		}
	}()
	select {
	case receivedReply := <-lc.responseChannels[nonce]:
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
	return nil
}

func (lc *litAfClient) ReceiveLoop() {
	for {
		msg := make([]byte, 1<<24)
		//	log.Printf("read message from %x\n", l.RemoteLNId)
		n, err := lc.lnconn.Read(msg)
		if err != nil {
			lc.lnconn.Close()
			panic(err)
		}
		msg = msg[:n]
		// We only care about RPC responses
		if msg[0] == lnutil.MSGID_REMOTE_RPCRESPONSE {
			response, err := lnutil.NewRemoteControlRpcResponseMsgFromBytes(msg, 0)
			if err != nil {
				panic(err)
			}

			responseChan, ok := lc.responseChannels[response.Idx]
			if ok {
				select {
				case responseChan <- response:
				default:
				}
				delete(lc.responseChannels, response.Idx)
			}

		}
	}

}
