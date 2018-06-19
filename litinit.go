package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mit-dci/lit/lnutil"
	nat "github.com/mit-dci/lit/nat"
)

// createDefaultConfigFile creates a config file  -- only call this if the
// config file isn't already there
func createDefaultConfigFile(destinationPath string) error {

	dest, err := os.OpenFile(filepath.Join(destinationPath, defaultConfigFilename),
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
func litSetup(conf *config) *[32]byte {
	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.

	//	usageMessage := fmt.Sprintf("Use %s -h to show usage", "./lit")

	preconf := *conf
	preParser := newConfigParser(&preconf, flags.HelpFlag)
	_, err := preParser.ParseArgs(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	// Load config from file and parse
	parser := newConfigParser(conf, flags.Default)

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
	// TODO ... what's this do?
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

	if conf.UPnP && conf.NatPmp {
		log.Println("Currently both Upnp and NAT-PMP cannot be " +
			"enabled together, using UPnP")
	}

	// do UPnP port forwarding
	// right now we fatal if we aren't able to port forward via upnp
	// a question though is whether we should continue connceting without
	// port forwarding if the user has explicitly told us so.
	if conf.UPnP {
		err := nat.SetupUpnp(conf.Rpcport)
		if err != nil {
			fmt.Printf("Unable to setup Upnp %v\n", err)
			log.Fatal(err)
		}
		log.Println("Forwarded port via UPnP")
		return key
		// don't go down further because in case both upnp and natpmp
		// are specified, we want upnp to take precedence.
	}

	if conf.NatPmp {
		discoveryTimeout := time.Duration(10 * time.Second)
		//log.Println("welcome to natpmp world")
		_, err := nat.SetupPmp(discoveryTimeout, conf.Rpcport)
		if err != nil {
			err := fmt.Errorf("Unable to discover a "+
				"NAT-PMP enabled device on the local "+
				"network: %v", err)
			log.Fatal(err)
		}

		// no need to return here since there's nothing after this
	}
	return key
}
