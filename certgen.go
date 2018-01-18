package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"time"
)

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

func GenCert() error {
	// generate cert
	if !pathExists("certs") {
		log.Println("Creating cert directort")
		err := os.Mkdir("certs", 0775)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	if !pathExists("certs/server.key") {
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

func genCertHandler(name string) error {

	/*
	Template copyright 2014, Jason Woods
	Script adapted from https://gist.github.com/glennwiz/74b01bc3dc916bdd2446
	*/

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
