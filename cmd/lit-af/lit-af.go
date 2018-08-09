package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/mit-dci/lit/btcutil/btcec"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
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
	remote        string
	port          uint16
	addr          string
	litHomeDir    string
	lndcRpcClient *litrpc.LndcRpcClient
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
	var err error

	lc := new(litAfClient)
	setConfig(lc)

	// create home directory if it does not exist
	_, err = os.Stat(lc.litHomeDir)
	if os.IsNotExist(err) {
		os.Mkdir(lc.litHomeDir, 0700)
	}

	if litrpc.LndcRpcCanConnectLocallyWithHomeDir(lc.litHomeDir) {
		lc.lndcRpcClient, err = litrpc.NewLocalLndcRpcClientWithHomeDirAndPort(lc.litHomeDir, uint32(lc.port))
		if err != nil {
			log.Fatal(err.Error())
		}
	} else {
		if !lnutil.LitAdrOK(lc.addr) {
			log.Fatal("Since you are remotely connecting to lit, you need to specify the node's LN address using the -addr parameter")
		}

		keyFilePath := filepath.Join(lc.litHomeDir, "lit-af-key.hex")
		privKey, err := lnutil.ReadKeyFile(keyFilePath)
		if err != nil {
			log.Fatal(err.Error())
		}
		key, _ := btcec.PrivKeyFromBytes(btcec.S256(), privKey[:])
		adr := fmt.Sprintf("%s@%s:%d", lc.addr, lc.remote, lc.port)
		lc.lndcRpcClient, err = litrpc.NewLndcRpcClient(adr, key)
		if err != nil {
			log.Fatal(err.Error())
		}
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
	return lc.lndcRpcClient.Call(serviceMethod, args, reply)
}
