package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mit-dci/lit/lnutil"
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

// helper function to see if a given path exists or not
func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// GenCert() creates ad directory, and appropriate server and client keys/certs
func GenCert() error {
	// generate cert
	if !pathExists("certs") {
		log.Println("Creating certs directorty")
		err := os.Mkdir("certs", 0775)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	if !pathExists("certs/server.key") {
		// if one is deleted, doesn't make sense to have the other
		log.Println("Generating server cert")
		err := genCertHandler("server")
		if err != nil {
			log.Println(err)
			return err
		}
	}

	if !pathExists("certs/client.key") {
		log.Println("Generating client cert")
		err := genCertHandler("client")
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

// genCertHandler creates a .pem and .key file for use by the listener and client respectively
// Template adapted from https://gist.github.com/glennwiz/74b01bc3dc916bdd2446

func genCertHandler(name string) error {

	var err error

	cert := x509.Certificate{
		Subject: pkix.Name{
			Organization: []string{"Lightning Network"},
		},
		NotBefore: time.Now(),

		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		IsCA: true,
	}

	cert.IPAddresses = append(cert.IPAddresses, net.ParseIP("127.0.0.1"))
	cert.NotAfter = cert.NotBefore.Add(time.Duration(365) * time.Hour * 24)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Println("Failed to generate private key:", err)
		return err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	cert.SerialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Println("Failed to generate serial number:", err)
		return err
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, &priv.PublicKey, priv)
	if err != nil {
		log.Println("Failed to create certificate:", err)
		return err
	}

	destPath := "certs/" + name + ".pem"
	certOut, err := os.Create(destPath)
	if err != nil {
		log.Println("Failed to open server.pem for writing:", err)
		return err
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyPath := "certs/" + name + ".key"
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Println("failed to open server.key for writing:", err)
		return err
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
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
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			log.Fatal(err)
		}
	}

	// Load config from file
	parser := newConfigParser(conf, flags.Default) //parse

	_, err = os.Stat(preconf.LitHomeDir) // create directory
	if err != nil {
		log.Println("Error while creating a directory")
	}
	if os.IsNotExist(err) {
		// first time the guy is running lit, lets set tn3 to true
		os.Mkdir(preconf.LitHomeDir, 0700)
		log.Println("Creating a new config file")
		err := createDefaultConfigFile(preconf.LitHomeDir) // Source of error
		if err != nil {
			log.Println("Error creating a default config file: %v\n")
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
	err = GenCert()
	if err != nil {
		log.Fatal(err)
	}
	return key
}
