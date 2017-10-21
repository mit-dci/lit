package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/litbamf"
	"github.com/mit-dci/lit/litrpc"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"
)

type config struct { // define a struct for usage with go-flags
	Tn3host     string `long:"tn3" description:"Connect to bitcoin testnet3."`
	Bc2host     string `long:"bc2" description:"bc2 full node."`
	Lt4host     string `long:"lt4" description:"Connect to litecoin testnet4."`
	Reghost     string `long:"reg" description:"Connect to bitcoin regtest."`
	Litereghost string `long:"litereg" description:"Connect to litecoin regtest."`
	Tvtchost    string `long:"tvtc" description:"Connect to Vertcoin test node."`
	Vtchost     string `long:"vtc" description:"Connect to Vertcoin."`
	LitHomeDir  string `long:"dir" description:"Specify Home Directory of lit as an absolute path."`
	TrackerURL  string `long:"tracker" description:"LN address tracker URL http|https://host:port"`
	ConfigFile  string

	ReSync  bool `short:"r" long:"reSync" description:"Resync from the given tip."`
	Tower   bool `long:"tower" description:"Watchtower: Run a watching node"`
	Hard    bool `short:"t" long:"hard" description:"Flag to set networks."`
	Verbose bool `short:"v" long:"verbose" description:"Set verbosity to true."`

	Rpcport uint16 `short:"p" long:"rpcport" description:"Set RPC port to connect to"`

	Params *coinparam.Params
}

var (
	defaultLitHomeDirName = os.Getenv("HOME") + "/.lit"
	defaultTrackerURL     = "http://ni.media.mit.edu:46580"
	defaultKeyFileName    = "privkey.hex"
	defaultConfigFilename = "lit.conf"
	defaultHomeDir        = os.Getenv("HOME")
	defaultConfigFile     = filepath.Join(os.Getenv("HOME"), "/.lit/lit.conf")
	defaultRpcport        = uint16(8001)
)

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// newConfigParser returns a new command line flags parser.
func newConfigParser(conf *config, options flags.Options) *flags.Parser {
	parser := flags.NewParser(conf, options)
	return parser
}

func linkWallets(node *qln.LitNode, key *[32]byte, conf *config) error {
	// for now, wallets are linked to the litnode on startup, and
	// can't appear / disappear while it's running.  Later
	// could support dynamically adding / removing wallets

	// order matters; the first registered wallet becomes the default

	var err error
	// try regtest
	if conf.Reghost != "" {
		p := &coinparam.RegressionNetParams
		if !strings.Contains(conf.Reghost, ":") {
			conf.Reghost = conf.Reghost + ":" + p.DefaultPort
		}
		fmt.Printf("reg: %s\n", conf.Reghost)
		err = node.LinkBaseWallet(key, 120, conf.ReSync, conf.Tower, conf.Reghost, p)
		if err != nil {
			return err
		}
	}
	// try testnet3
	if conf.Tn3host != "" {
		p := &coinparam.TestNet3Params
		if !strings.Contains(conf.Tn3host, ":") {
			conf.Tn3host = conf.Tn3host + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(
			key, 1210000, conf.ReSync, conf.Tower,
			conf.Tn3host, p)
		if err != nil {
			return err
		}
	}
	// try litecoin regtest
	if conf.Litereghost != "" {
		p := &coinparam.LiteRegNetParams
		if !strings.Contains(conf.Litereghost, ":") {
			conf.Litereghost = conf.Litereghost + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(key, 120, conf.ReSync, conf.Tower, conf.Litereghost, p)
		if err != nil {
			return err
		}
	}

	// try litecoin testnet4
	if conf.Lt4host != "" {
		p := &coinparam.LiteCoinTestNet4Params
		if !strings.Contains(conf.Lt4host, ":") {
			conf.Lt4host = conf.Lt4host + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.ReSync, conf.Tower,
			conf.Lt4host, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin testnet
	if conf.Tvtchost != "" {
		p := &coinparam.VertcoinTestNetParams
		if !strings.Contains(conf.Tvtchost, ":") {
			conf.Tvtchost = conf.Tvtchost + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(
			key, 0, conf.ReSync, conf.Tower,
			conf.Tvtchost, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin mainnet
	if conf.Vtchost != "" {
		p := &coinparam.VertcoinParams
		if !strings.Contains(conf.Vtchost, ":") {
			conf.Vtchost = conf.Vtchost + ":" + p.DefaultPort
		}
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.ReSync, conf.Tower,
			conf.Vtchost, p)
		if err != nil {
			return err
		}

	}
	return nil
}

func main() {

	conf := config{
		LitHomeDir: defaultLitHomeDirName,
		ConfigFile: defaultConfigFile,
		Rpcport:    defaultRpcport,
		TrackerURL: defaultTrackerURL,
	}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.
	preconf := conf
	preParser := newConfigParser(&preconf, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil { // if there is some sort of error while parsing the CLI arguments
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
			log.Fatal(err)
			return
			// return nil, nil, err
		}
	}

	// appName := filepath.Base(os.Args[0])
	// appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	// usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)
	// if preconf.ShowVersion {
	// 	fmt.Println(appName, "version", version())
	// 	os.Exit(0)
	// }

	// Load additional config from file
	var configFileError error
	parser := newConfigParser(&conf, flags.Default) // Single line command to read all the CLI params passed

	// creates a directory in the absolute sense
	if _, err := os.Stat(preconf.LitHomeDir); os.IsNotExist(err) {
		os.Mkdir(preconf.LitHomeDir, 0700)
		fmt.Println("Creating a new config file")
		err1 := createDefaultConfigFile(preconf.LitHomeDir) // Source of error
		if err1 != nil {
			fmt.Fprintf(os.Stderr, "Error creating a "+
				"default config file: %v\n", err)
		}
	}

	if err != nil {
		fmt.Println("Error while creating a directory")
		fmt.Println(err)
	}

	if !(preconf.ConfigFile != defaultConfigFile) {
		// passing works fine.
		// fmt.Println("Watch out")
		// fmt.Println(filepath.Join(preconf.LitHomeDir))
		if _, err := os.Stat(filepath.Join(filepath.Join(preconf.LitHomeDir), "lit.conf")); os.IsNotExist(err) {
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("Creating a new config file")
			err1 := createDefaultConfigFile(filepath.Join(preconf.LitHomeDir)) // Source of error
			if err1 != nil {
				fmt.Fprintf(os.Stderr, "Error creating a "+
					"default config file: %v\n", err)
			}
		}
		preconf.ConfigFile = filepath.Join(filepath.Join(preconf.LitHomeDir), "lit.conf")
		err := flags.NewIniParser(parser).ParseFile(preconf.ConfigFile) // lets parse the config file provided, if any
		if err != nil {
			if _, ok := err.(*os.PathError); !ok {
				fmt.Fprintf(os.Stderr, "Error parsing config "+
					"file: %v\n", err)
				// fmt.Fprintln(os.Stderr, usageMessage)
				log.Fatal(err)
				// return nil, nil, err
			}
			configFileError = err
		}
	}

	// Parse command line options again to ensure they take precedence.
	remainingArgs, err := parser.Parse() // no extra work, free overloading.
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			// fmt.Fprintln(os.Stderr, usageMessage)
		}
		log.Fatal(err)
		// return nil, nil, err
	}

	if configFileError != nil {
		fmt.Printf("%v", configFileError)
	}

	if remainingArgs != nil {
		//fmt.Printf("%v", remainingArgs)
	}

	logFilePath := filepath.Join(conf.LitHomeDir, "lit.log")

	logfile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer logfile.Close()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	if conf.Verbose {
		logOutput := io.MultiWriter(os.Stdout, logfile)
		log.SetOutput(logOutput)
	} else {
		log.SetOutput(logfile)
	}

	// Allow node with no linked wallets, for testing.
	// TODO Should update tests and disallow nodes without wallets later.
	//	if conf.Tn3host == "" && conf.Lt4host == "" && conf.Reghost == "" {
	//		log.Fatal("error: no network specified; use -tn3, -reg, -lt4")
	//	}

	// Keys: the litNode, and wallits, all get 32 byte keys.
	// Right now though, they all get the *same* key.  For lit as a single binary
	// now, all using the same key makes sense; could split up later.

	keyFilePath := filepath.Join(conf.LitHomeDir, defaultKeyFileName)

	// read key file (generate if not found)
	key, err := lnutil.ReadKeyFile(keyFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Setup LN node.  Activate Tower if in hard mode.
	// give node and below file pathof lit home directoy
	node, err := qln.NewLitNode(key, conf.LitHomeDir, conf.TrackerURL)
	if err != nil {
		log.Fatal(err)
	}

	// node is up; link wallets based on args
	err = linkWallets(node, key, &conf)
	if err != nil {
		log.Fatal(err)
	}

	rpcl := new(litrpc.LitRPC)
	rpcl.Node = node
	rpcl.OffButton = make(chan bool, 1)

	litrpc.RPCListen(rpcl, conf.Rpcport)
	litbamf.BamfListen(conf.Rpcport, conf.LitHomeDir)

	<-rpcl.OffButton
	fmt.Printf("Got stop request\n")
	time.Sleep(time.Second)

	return
	// New directory being created over at PWD
	// conf file being created at /
}

func createDefaultConfigFile(destinationPath string) error {

	// We assume sample config file path is same as binary TODO: change to ~/.lit/config/
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	sampleConfigPath := filepath.Join(path, defaultConfigFilename)

	// We generate a random user and password
	randomBytes := make([]byte, 20)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return err
	}
	generatedRPCUser := base64.StdEncoding.EncodeToString(randomBytes)

	_, err = rand.Read(randomBytes)
	if err != nil {
		return err
	}
	generatedRPCPass := base64.StdEncoding.EncodeToString(randomBytes)

	src, err := os.Open(sampleConfigPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.OpenFile(filepath.Join(destinationPath, defaultConfigFilename),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dest.Close()

	// We copy every line from the sample config file to the destination,
	// only replacing the two lines for rpcuser and rpcpass
	reader := bufio.NewReader(src)
	for err != io.EOF {
		var line string
		line, err = reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		if strings.Contains(line, "rpcuser=") {
			line = "rpcuser=" + generatedRPCUser + "\n"
		} else if strings.Contains(line, "rpcpass=") {
			line = "rpcpass=" + generatedRPCPass + "\n"
		}

		if _, err := dest.WriteString(line); err != nil {
			return err
		}
	}
	return nil
}
