package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	flags "github.com/jessevdk/go-flags"
	"github.com/mit-dci/lit/crypto/koblitz"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
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

const (
	litHomeDirName     = ".lit"
	historyFilename    = "lit-af.history"
	defaultKeyFileName = "privkey.hex"
)

type Command struct {
	Format           string
	Description      string
	ShortDescription string
}

type litAfClient struct {
	RPCClient *litrpc.LndcRpcClient
}

type litAfConfig struct {
	Con      string `long:"con" description:"host to connect to in the form of [<lnadr>@][<host>][:<port>]"`
	Dir      string `long:"dir" description:"directory to save settings"`
	Tracker  string `long:"tracker" description:"service to use for looking up node addresses"`
	LogLevel []bool `short:"v" description:"Set verbosity level to verbose (-v), very verbose (-vv) or very very verbose (-vvv)"`
}

var (
	defaultCon     = "2448"
	defaultDir     = filepath.Join(os.Getenv("HOME"), litHomeDirName)
	defaultTracker = "http://hubris.media.mit.edu:46580"
)

// newConfigParser returns a new command line flags parser.
func newConfigParser(conf *litAfConfig, options flags.Options) *flags.Parser {
	parser := flags.NewParser(conf, options)
	return parser
}

func (lc *litAfClient) litAfSetup(conf litAfConfig) error {

	var err error
	// create home directory if it does not exist
	_, err = os.Stat(defaultDir)
	if os.IsNotExist(err) {
		os.Mkdir(defaultDir, 0700)
	}

	preParser := newConfigParser(&conf, flags.HelpFlag)
	_, err = preParser.ParseArgs(os.Args) // parse the cli
	if err != nil {
		logging.Fatal(err)
	}
	logLevel := 0
	if len(conf.LogLevel) == 1 { // -v
		logLevel = 1
	} else if len(conf.LogLevel) == 2 { // -vv
		logLevel = 2
	} else if len(conf.LogLevel) >= 3 { // -vvv
		logLevel = 3
	}
	logging.SetLogLevel(logLevel) // defaults to zero

	// we don't know whether the passed address is a remotePKH or a remotePK
	// so we need to detect that here and then take steps accordingly
	// so the first part involved here would be either dealing with raw pubkeys or
	// dealing with pk hashes
	// another question here is how do we deal with connecting to other nodes?
	// if we need that in, we need to hcange our overall architecture to host
	// pks as well as pk hashes
	adr, host, port := lnutil.ParseAdrStringWithPort(conf.Con)
	// now we've split the address, check if pkh, if not, route straight to noise_xk
	if len(adr) == 44 {
		// remote PKH, do standard stuff
		fmt.Printf("Adr: %s, Host: %s, Port: %d\n", adr, host, port)
		if litrpc.LndcRpcCanConnectLocallyWithHomeDir(defaultDir) && adr == "" && (host == "localhost" || host == "127.0.0.1") {

			lc.RPCClient, err = litrpc.NewLocalLndcRpcClientWithHomeDirAndPort(defaultDir, port)
			if err != nil {
				logging.Fatal(err.Error())
			}
		} else {
			if !lnutil.LitAdrOK(adr) {
				logging.Fatal("lit address passed in -con parameter is not valid")
			}

			keyFilePath := filepath.Join(defaultDir, "lit-af-key.hex")
			privKey, err := lnutil.ReadKeyFile(keyFilePath)
			if err != nil {
				logging.Fatal(err.Error())
			}
			key, _ := koblitz.PrivKeyFromBytes(koblitz.S256(), privKey[:])
			pubkey := key.PubKey().SerializeCompressed() // this is in bytes
			fmt.Printf("The pubkey of this lit-af instance is: %s\n", hex.EncodeToString(pubkey))
			var temp [33]byte
			copy(temp[:], pubkey[:33])
			fmt.Printf("The pkh of this lit-af instance is: %s\n", lnutil.LitAdrFromPubkey(temp))
			if adr != "" && strings.HasPrefix(adr, "ln1") && host == "" {
				ipv4, _, err := lnutil.Lookup(adr, conf.Tracker, "")
				if err != nil {
					logging.Fatalf("Error looking up address on the tracker: %s", err)
				} else {
					adr = fmt.Sprintf("%s@%s", adr, ipv4)
				}
			} else {
				adr = fmt.Sprintf("%s@%s:%d", adr, host, port)
			}

			lc.RPCClient, err = litrpc.NewLndcRpcClient(adr, key)
			if err != nil {
				logging.Fatal(err.Error())
			}
		}
	} else if len(adr) == 66 {
		// remote PK, don't convert to remotePKH
		// we have a pubkey, we should't bother checking addresses
		// this is definitively a non-lit instance, so don't check for lit stuff here
		keyFilePath := filepath.Join(defaultDir, "lit-af-key.hex")
		privKey, err := lnutil.ReadKeyFile(keyFilePath)
		if err != nil {
			logging.Fatal(err.Error())
		}
		key, _ := koblitz.PrivKeyFromBytes(koblitz.S256(), privKey[:])
		pubkey := key.PubKey().SerializeCompressed() // this is in bytes
		fmt.Printf("The pubkey of this lit-af instance is: %s\n", hex.EncodeToString(pubkey))
		var temp [33]byte
		copy(temp[:], pubkey[:33])
		fmt.Printf("The pkh of this lit-af instance is: %s\n", lnutil.LitAdrFromPubkey(temp))
		// so this is a non lit node and hence, we don't need to lookup on the tracker (since we already have the pubkey)

		// initialize a new rpc instance with noise_xk based authentication here
		adr = fmt.Sprintf("%s@%s:%d", adr, host, port)
		err = litrpc.NewOtherRpcClient(adr, key)
		if err != nil {
			logging.Fatal(err.Error())
		}
	}
	return nil
}

// for now just testing how to connect and get messages back and forth
func main() {

	var err error
	lc := new(litAfClient)
	conf := litAfConfig{
		Con:     defaultCon,
		Dir:     defaultDir,
		Tracker: defaultTracker,
	}
	err = lc.litAfSetup(conf) // setup lit-af to start
	if err != nil {
		logging.Error(err)
		return
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       lnutil.Prompt("lit-af") + lnutil.White("# "),
		HistoryFile:  filepath.Join(defaultDir, historyFilename),
		AutoComplete: lc.NewAutoCompleter(),
	})
	if err != nil {
		logging.Fatal(err)
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
			logging.Fatal(err)
		}
	}
}

func (lc *litAfClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return lc.RPCClient.Call(serviceMethod, args, reply)
}
