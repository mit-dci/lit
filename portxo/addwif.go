package portxo

import (
	"bytes"
	"fmt"

	"github.com/mit-dci/lit/logging"

	"github.com/mit-dci/lit/btcutil"
	"github.com/mit-dci/lit/coinparam"
)

func (u *PorTxo) AddWIF(w btcutil.WIF) error {
	var err error

	// check that WIF and utxo net ID match

	//... which is super annoying because the wif netID is not capitalized.
	// so we'll capitalize it and be able to check just by matching bytes.

	//	if u.NetID != w.netID {
	//		return fmt.Errorf("utxo and wif key networks don't match")
	//	}

	// if TxoMode is set, check that compressed / uncompressed match
	if u.Mode != TxoUnknownMode {
		if w.CompressPubKey && u.Mode&FlagTxoCompressed == 0 {
			return fmt.Errorf("utxo %s uncompressed, but WIF key is compressed",
				u.Op.String())
		}
		if !w.CompressPubKey && u.Mode&FlagTxoCompressed != 0 {
			return fmt.Errorf("utxo %s compressed, but WIF key is uncompressed",
				u.Op.String())
		}
	}

	// if script exists and mode is set to PKH, check if pubkey matches script
	if u.PkScript != nil && (u.Mode&FlagTxoPubKeyHash != 0) {
		// just check testnet and mainnet for now, can add more later.
		// annoying that WIF can't return params... it should...

		adr := new(btcutil.AddressPubKeyHash)

		if w.IsForNet(&coinparam.TestNet3Params) {
			// generate address from wif key
			adr, err = btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(w.SerializePubKey()), &coinparam.TestNet3Params)
			if err != nil {
				return err
			}
		} else { // assume mainnet
			// generate address from wif key
			adr, err = btcutil.NewAddressPubKeyHash(
				btcutil.Hash160(w.SerializePubKey()), &coinparam.BitcoinParams)
			if err != nil {
				return err
			}
		}
		if len(u.PkScript) != 25 {
			return fmt.Errorf("pkh utxo script %d bytes (not 25)", len(u.PkScript))
		}
		if !bytes.Equal(adr.ScriptAddress(), u.PkScript[3:23]) {
			logging.Errorf("utxopk:\t%x\nwifpk:\t%x\n",
				u.PkScript[3:23], adr.ScriptAddress())
			return fmt.Errorf("utxo and wif addresses don't match")
		}
	}

	// everything OK, copy in private key
	copy(u.PrivKey[:], w.PrivKey.Serialize())

	return nil
}
