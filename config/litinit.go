package config

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"github.com/mit-dci/lit/coinparam"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/qln"
)

// createDefaultConfigFile creates a config file  -- only call this if the
// config file isn't already there
func createDefaultConfigFile(destinationPath string) error {

	dest, err := os.OpenFile(filepath.Join(destinationPath, DefaultConfigFilename),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer dest.Close()

	writer := bufio.NewWriter(dest)
	defaultArgs := []byte("tn3=1")
	_, err = writer.Write(defaultArgs)
	if err != nil {
		return err
	}
	writer.Flush()
	return nil
}

// litSetup performs most of the setup when lit is run, such as setting
// configuration variables, reading in key data, reading and creating files if
// they're not yet there.  It takes in a config, and returns a key.
// (maybe add the key to the config?
func LitSetup(conf *Config) *[32]byte {
	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.

	//	usageMessage := fmt.Sprintf("Use %s -h to show usage", "./lit")

	preconf := *conf
	preParser := NewConfigParser(&preconf, flags.HelpFlag)
	_, err := preParser.ParseArgs(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	// Load config from file and parse
	parser := NewConfigParser(conf, flags.Default)

	// create home directory
	_, err = os.Stat(preconf.LitHomeDir)
	if err != nil {
		log.Println("Error while creating a directory")
	}
	if os.IsNotExist(err) {
		// first time the guy is running lit, lets set tn3 to true
		os.Mkdir(preconf.LitHomeDir, 0700)
		log.Println("Creating a new config file")
		err := createDefaultConfigFile(preconf.LitHomeDir) // Source of error
		if err != nil {
			log.Printf("Error creating a default config file: %v", preconf.LitHomeDir)
			log.Fatal(err)
		}
	}

	if _, err := os.Stat(filepath.Join(filepath.Join(preconf.LitHomeDir), "lit.conf")); os.IsNotExist(err) {
		// if there is no config file found over at the directory, create one
		if err != nil {
			log.Println(err)
		}
		log.Println("Creating a new config file")
		err := createDefaultConfigFile(filepath.Join(preconf.LitHomeDir)) // Source of error
		if err != nil {
			log.Fatal(err)
		}
	}

	preconf.ConfigFile = filepath.Join(filepath.Join(preconf.LitHomeDir), "lit.conf")
	// lets parse the config file provided, if any
	err = flags.NewIniParser(parser).ParseFile(preconf.ConfigFile)
	if err != nil {
		_, ok := err.(*os.PathError)
		if !ok {
			log.Fatal(err)
		}
	}
	// Parse command line options again to ensure they take precedence.
	_, err = parser.ParseArgs(os.Args) // returns invalid flags
	if err != nil {
		log.Fatal(err)
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

	keyFilePath := filepath.Join(conf.LitHomeDir, DefaultKeyFileName)

	// read key file (generate if not found)
	key, err := lnutil.ReadKeyFile(keyFilePath)
	if err != nil {
		log.Fatal(err)
	}

	return key
}

func LinkWallets(node *qln.LitNode, key *[32]byte, conf *Config) error {
	// for now, wallets are linked to the litnode on startup, and
	// can't appear / disappear while it's running.  Later
	// could support dynamically adding / removing wallets

	// order matters; the first registered wallet becomes the default

	var err error
	// try regtest
	if !lnutil.NopeString(conf.Reghost) {
		p := &coinparam.RegressionNetParams
		log.Printf("reg: %s\n", conf.Reghost)
		err = node.LinkBaseWallet(key, 120, conf.ReSync, conf.Tower, conf.Reghost, p)
		if err != nil {
			return err
		}
	}
	// try testnet3
	if !lnutil.NopeString(conf.Tn3host) {
		p := &coinparam.TestNet3Params
		err = node.LinkBaseWallet(
			key, 1256000, conf.ReSync, conf.Tower,
			conf.Tn3host, p)
		if err != nil {
			return err
		}
	}
	// try litecoin regtest
	if !lnutil.NopeString(conf.Litereghost) {
		p := &coinparam.LiteRegNetParams
		err = node.LinkBaseWallet(key, 120, conf.ReSync, conf.Tower, conf.Litereghost, p)
		if err != nil {
			return err
		}
	}
	// try litecoin testnet4
	if !lnutil.NopeString(conf.Lt4host) {
		p := &coinparam.LiteCoinTestNet4Params
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.ReSync, conf.Tower,
			conf.Lt4host, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin testnet
	if !lnutil.NopeString(conf.Tvtchost) {
		p := &coinparam.VertcoinTestNetParams
		err = node.LinkBaseWallet(
			key, 25000, conf.ReSync, conf.Tower,
			conf.Tvtchost, p)
		if err != nil {
			return err
		}
	}
	// try vertcoin mainnet
	if !lnutil.NopeString(conf.Vtchost) {
		p := &coinparam.VertcoinParams
		err = node.LinkBaseWallet(
			key, p.StartHeight, conf.ReSync, conf.Tower,
			conf.Vtchost, p)
		if err != nil {
			return err
		}

	}
	return nil
}
