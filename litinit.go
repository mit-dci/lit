package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
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
func litSetup(conf *litConfig) *[32]byte {
	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.

	//	usageMessage := fmt.Sprintf("Use %s -h to show usage", "./lit")

	preParser := newConfigParser(conf, flags.HelpFlag)
	_, err := preParser.ParseArgs(os.Args)
	if err != nil {
		logging.Fatal(err)
	}

	// Load config from file and parse
	parser := newConfigParser(conf, flags.Default)

	// set default log level here
	logging.SetLogLevel(defaultLogLevel)
	// create home directory
	_, err = os.Stat(conf.LitHomeDir)
	if err != nil {
		logging.Errorf("Error while creating a directory")
	}
	if os.IsNotExist(err) {
		// first time the guy is running lit, lets set tn3 to true
		os.Mkdir(conf.LitHomeDir, 0700)
		logging.Infof("Creating a new config file")
		err := createDefaultConfigFile(conf.LitHomeDir) // Source of error
		if err != nil {
			fmt.Printf("Error creating a default config file: %v", conf.LitHomeDir)
			logging.Fatal(err)
		}
	}

	if _, err := os.Stat(filepath.Join(filepath.Join(conf.LitHomeDir), "lit.conf")); os.IsNotExist(err) {
		// if there is no config file found over at the directory, create one
		if err != nil {
			fmt.Println(err)
		}
		logging.Infof("Creating a new config file")
		err := createDefaultConfigFile(filepath.Join(conf.LitHomeDir)) // Source of error
		if err != nil {
			logging.Fatal(err)
		}
	}

	conf.ConfigFile = filepath.Join(filepath.Join(conf.LitHomeDir), "lit.conf")
	// lets parse the config file provided, if any
	err = flags.NewIniParser(parser).ParseFile(conf.ConfigFile)
	if err != nil {
		_, ok := err.(*os.PathError)
		if !ok {
			logging.Fatal(err)
		}
	}
	// Parse command line options again to ensure they take precedence.
	_, err = parser.ParseArgs(os.Args) // returns invalid flags
	if err != nil {
		logging.Fatal(err)
	}

	logFilePath := filepath.Join(conf.LitHomeDir, "lit.log")
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	logging.SetLogFile(logFile)

	// Log Levels:
	// 5: DebugLevel prints Panics, Fatals, Errors, Warnings, Infos and Debugs
	// 4: InfoLevel  prints Panics, Fatals, Errors, Warnings and Info
	// 3: WarnLevel  prints Panics, Fatals, Errors and Warnings
	// 2: ErrorLevel prints Panics, Fatals and Errors
	// 1: FatalLevel prints Panics, Fatals
	// 0: PanicLevel prints Panics
	// Default is level 3
	// Code for tagging logs:
	// Debug -> Useful debugging information
	// Info  -> Something noteworthy happened
	// Warn  -> You should probably take a look at this
	// Error -> Something failed but I'm not quitting
	// Fatal -> Bye

	// TODO ... what's this do?
	defer logFile.Close()

	logLevel := -1
	if len(conf.LogLevel) == 1 { // -v
		logLevel = 1
	} else if len(conf.LogLevel) == 2 { // -vv
		logLevel = 2
	} else if len(conf.LogLevel) >= 3 {
		logLevel = 3
	}
	logging.SetLogLevel(logLevel)

	// Allow node with no linked wallets, for testing.
	// TODO Should update tests and disallow nodes without wallets later.
	// Keys: the litNode, and wallits, all get 32 byte keys.
	// Right now though, they all get the *same* key.  For lit as a single binary
	// now, all using the same key makes sense; could split up later.

	keyFilePath := filepath.Join(conf.LitHomeDir, defaultKeyFileName)

	// read key file (generate if not found)
	key, err := lnutil.ReadKeyFile(keyFilePath)
	if err != nil {
		logging.Fatal(err)
	}

	return key
}
