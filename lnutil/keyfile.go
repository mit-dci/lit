package lnutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/howeyc/gopass"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/scrypt"
)

/* should switch to the codahale/chacha20poly1305 package.  Or whatever we're
using for LNDC; otherwise it's 2 almost-the-same stream ciphers to import */

// warning! look at those imports! crypto! hopefully this works!

/* on-disk stored keys are 32bytes.  This is good for ed25519 private keys,
for seeds for bip32, for individual secp256k1 priv keys, and so on.
32 bytes is enough for anyone.
If you want fewer bytes, put some zeroes at the end */

// LoadKeyFromFileInteractive opens the file 'filename' and presents a
// keyboard prompt for the passphrase to decrypt it.  It returns the
// key if decryption works, or errors out.
func LoadKeyFromFileInteractive(filename string) (*[32]byte, error) {
	a, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	if a.Size() < 80 { // there can't be a password...
		return LoadKeyFromFileArg(filename, nil)
	}
	log.Printf("passphrase: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		return nil, err
	}
	log.Printf("\n")
	return LoadKeyFromFileArg(filename, pass)
}

// LoadKeyFromFileArg opens the file and returns the key.  If the key is
// unencrypted it will ignore the password argument.
func LoadKeyFromFileArg(filename string, pass []byte) (*[32]byte, error) {
	priv32 := new([32]byte)
	keyhex, err := ioutil.ReadFile(filename)
	if err != nil {
		return priv32, err
	}
	keyhex = []byte(strings.TrimSpace(string(keyhex)))
	enckey, err := hex.DecodeString(string(keyhex))
	if err != nil {
		return priv32, err
	}

	if len(enckey) == 32 { // UNencrypted key, length 32
		log.Printf("WARNING!! Key file not encrypted!!\n")
		log.Printf("Anyone who can read the key file can take everything!\n")
		log.Printf("You should start over and use a good passphrase!\n")
		copy(priv32[:], enckey[:])
		return priv32, nil
	}
	// enckey should be 72 bytes.  24 for scrypt salt/box nonce,
	// 16 for box auth
	if len(enckey) != 72 {
		return priv32, fmt.Errorf("Key length error for %s ", filename)
	}
	// enckey is actually encrypted, get derived key from pass and salt
	// first extract salt
	salt := new([24]byte)      // salt (also nonce for secretbox)
	dk32 := new([32]byte)      // derived key array
	copy(salt[:], enckey[:24]) // first 24 bytes are scrypt salt/box nonce

	dk, err := scrypt.Key(pass, salt[:], 16384, 8, 1, 32) // derive key
	if err != nil {
		return priv32, err
	}
	copy(dk32[:], dk[:]) // copy into fixed size array

	// nonce for secretbox is the same as scrypt salt.  Seems fine.  Really.
	priv, worked := secretbox.Open(nil, enckey[24:], salt, dk32)
	if worked != true {
		return priv32, fmt.Errorf("Decryption failed for %s ", filename)
	}
	copy(priv32[:], priv[:]) //copy decrypted private key into array

	priv = nil // this probably doesn't do anything but... eh why not
	return priv32, nil
}

// saves a 32 byte key to file, prompting for passphrase.
// if user enters empty passphrase (hits enter twice), will be saved
// in the clear.
func SaveKeyToFileInteractive(filename string, priv32 *[32]byte) error {
	var match bool
	var err error
	var pass1, pass2 []byte
	for match != true {
		log.Printf("passphrase: ")
		pass1, err = gopass.GetPasswd()
		if err != nil {
			return err
		}
		log.Printf("repeat passphrase: ")
		pass2, err = gopass.GetPasswd()
		if err != nil {
			return err
		}
		if string(pass1) == string(pass2) {
			match = true
		} else {
			log.Printf("user input error.  Try again gl hf dd.\n")
		}
	}
	log.Printf("\n")
	return SaveKeyToFileArg(filename, priv32, pass1)
}

// saves a 32 byte key to a file, encrypting with pass.
// if pass is nil or zero length, doesn't encrypt and just saves in hex.
func SaveKeyToFileArg(filename string, priv32 *[32]byte, pass []byte) error {
	if len(pass) == 0 { // zero-length pass, save unencrypted
		keyhex := fmt.Sprintf("%x\n", priv32[:])
		err := ioutil.WriteFile(filename, []byte(keyhex), 0600)
		if err != nil {
			return err
		}
		log.Printf("WARNING!! Key file not encrypted!!\n")
		log.Printf("Anyone who can read the key file can take everything!\n")
		log.Printf("You should start over and use a good passphrase!\n")
		log.Printf("Saved unencrypted key at %s\n", filename)
		return nil
	}

	salt := new([24]byte) // salt for scrypt / nonce for secretbox
	dk32 := new([32]byte) // derived key from scrypt

	//get 24 random bytes for scrypt salt (and secretbox nonce)
	_, err := rand.Read(salt[:])
	if err != nil {
		return err
	}
	// next use the pass and salt to make a 32-byte derived key
	dk, err := scrypt.Key(pass, salt[:], 16384, 8, 1, 32)
	if err != nil {
		return err
	}
	copy(dk32[:], dk[:])

	enckey := append(salt[:], secretbox.Seal(nil, priv32[:], salt, dk32)...)
	//	enckey = append(salt, enckey...)
	keyhex := fmt.Sprintf("%x\n", enckey)

	err = ioutil.WriteFile(filename, []byte(keyhex), 0600)
	if err != nil {
		return err
	}
	log.Printf("Wrote encrypted key to %s\n", filename)
	return nil
}

// ReadKeyFile returns an 32 byte key from a file.
// If there's no file there, it'll make one.  If there's a password needed,
// it'll prompt for one.  One stop function.
func ReadKeyFile(filename string) (*[32]byte, error) {
	key32 := new([32]byte)
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// no key found, generate and save one
			log.Printf("No file %s, generating.\n", filename)

			_, err := rand.Read(key32[:])
			if err != nil {
				return nil, err
			}

			err = SaveKeyToFileInteractive(filename, key32)
			if err != nil {
				return nil, err
			}
		} else {
			// unknown error, crash
			log.Printf("unknown\n")
			return nil, err
		}
	}
	return LoadKeyFromFileInteractive(filename)
}
