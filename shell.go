package main

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"

	"github.com/mit-dci/lit/lndc"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/qln"
	"github.com/mit-dci/lit/uspv"
)

/* lit shell cooked in.  Switch to rpc shell soon */

// Shellparse parses user input and hands it to command functions if matching
func Shellparse(cmdslice []string) error {
	var err error
	var args []string
	cmd := cmdslice[0]
	if len(cmdslice) > 1 {
		args = cmdslice[1:]
	}
	if cmd == "exit" || cmd == "quit" {
		return fmt.Errorf("User exit")
	}
	// help gives you really terse help.  Just a list of commands.
	if cmd == "help" {
		err = Help(args)
		if err != nil {
			fmt.Printf("help error: %s\n", err)
		}
		return nil
	}
	// adr generates a new address and displays it
	if cmd == "adr" {
		err = Adr(args)
		if err != nil {
			fmt.Printf("adr error: %s\n", err)
		}
		return nil
	}
	if cmd == "fake" { // give yourself fake utxos.
		err = Fake(args)
		if err != nil {
			fmt.Printf("fake error: %s\n", err)
		}
		return nil
	}
	// bal shows the current set of utxos, addresses and score
	if cmd == "bal" {
		err = Bal(args)
		if err != nil {
			fmt.Printf("bal error: %s\n", err)
		}
		return nil
	}

	// send sends coins to the address specified
	if cmd == "send" {
		err = Send(args)
		if err != nil {
			fmt.Printf("send error: %s\n", err)
		}
		return nil
	}
	if cmd == "msend" {
		err = MSend(args)
		if err != nil {
			fmt.Printf("Msend error: %s\n", err)
		}
		return nil
	}
	if cmd == "rsend" {
		err = RSend(args)
		if err != nil {
			fmt.Printf("Rsend error: %s\n", err)
		}
		return nil
	}
	if cmd == "nsend" {
		err = NSend(args)
		if err != nil {
			fmt.Printf("Nsend error: %s\n", err)
		}
		return nil
	}

	if cmd == "fan" { // fan-out tx
		err = Fan(args)
		if err != nil {
			fmt.Printf("fan error: %s\n", err)
		}
		return nil
	}
	if cmd == "sweep" { // make lots of 1-in 1-out txs
		err = Sweep(args)
		if err != nil {
			fmt.Printf("sweep error: %s\n", err)
		}
		return nil
	}
	if cmd == "txs" { // show all txs
		err = Txs(args)
		if err != nil {
			fmt.Printf("txs error: %s\n", err)
		}
		return nil
	}
	if cmd == "con" { // connect to lnd host
		err = Con(args)
		if err != nil {
			fmt.Printf("con error: %s\n", err)
		}
		return nil
	}
	if cmd == "wcon" { // connect to watch tower
		err = WCon(args)
		if err != nil {
			fmt.Printf("wcon error: %s\n", err)
		}
		return nil
	}

	if cmd == "watch" { // connect to watch tower
		err = Watch(args)
		if err != nil {
			fmt.Printf("watch error: %s\n", err)
		}
		return nil
	}

	if cmd == "lis" { // listen for lnd peers
		err = Lis(args)
		if err != nil {
			fmt.Printf("lis error: %s\n", err)
		}
		return nil
	}

	// Peer to peer actions
	// send text message
	if cmd == "say" {
		err = Say(args)
		if err != nil {
			fmt.Printf("say error: %s\n", err)
		}
		return nil
	}
	// fund and create a new channel
	if cmd == "fund" {
		err = FundChannel(args)
		if err != nil {
			fmt.Printf("fund error: %s\n", err)
		}
		return nil
	}
	// push money in a channel away from you
	if cmd == "push" {
		err = Push(args)
		if err != nil {
			fmt.Printf("push error: %s\n", err)
		}
		return nil
	}
	// cooperateive close of a channel
	if cmd == "cclose" {
		err = CloseChannel(args)
		if err != nil {
			fmt.Printf("cclose error: %s\n", err)
		}
		return nil
	}
	if cmd == "break" {
		err = BreakChannel(args)
		if err != nil {
			fmt.Printf("break error: %s\n", err)
		}
		return nil
	}

	if cmd == "fix" {
		err = Resume(args)
		if err != nil {
			fmt.Printf("fix error: %s\n", err)
		}
		return nil
	}

	fmt.Printf("Command not recognized. type help for command list.\n")
	return nil
}

// Lis starts listening.  Takes args of port to listen on.
func Lis(args []string) error {
	portstring := ":2448"
	if len(args) > 0 {
		portstring = args[0]
		portstring = ":" + portstring
	}
	go Node.TCPListener(portstring)
	return nil
}

// connects to the watchtower
func WCon(args []string) error {
	var err error

	if len(args) == 0 {
		return fmt.Errorf("need: watch pubkeyhash@hostname:port")
	}

	newWatch, err := lndc.LnAddrFromString(args[0])
	if err != nil {
		return err
	}

	idPriv := Node.IdKey()
	Node.WatchCon = new(lndc.LNDConn)

	err = Node.WatchCon.Dial(
		idPriv, newWatch.NetAddr.String(), newWatch.Base58Adr.ScriptAddress())
	if err != nil {
		return err
	}

	// receiver is just the regular lndc.  Really they should be the same, just flag
	// some lndcs as watchers
	return nil
}

func Con(args []string) error {
	var err error

	if len(args) == 0 {
		return fmt.Errorf("need: con pubkeyhash@hostname:port")
	}

	newNode, err := lndc.LnAddrFromString(args[0])
	if err != nil {
		return err
	}

	idPriv := Node.IdKey()
	Node.RemoteCon = new(lndc.LNDConn)

	err = Node.RemoteCon.Dial(
		idPriv, newNode.NetAddr.String(), newNode.Base58Adr.ScriptAddress())
	if err != nil {
		return err
	}
	// store this peer
	_, err = Node.NewPeer(Node.RemoteCon.RemotePub)
	if err != nil {
		return err
	}

	idslice := btcutil.Hash160(Node.RemoteCon.RemotePub.SerializeCompressed())
	var newId [16]byte
	copy(newId[:], idslice[:16])
	go Node.LNDCReceiver(Node.RemoteCon, newId)

	return nil
}

// Say sends a text string
// For fun / testing.  Syntax: say hello world
func Say(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("you have to say something")
	}
	if Node.RemoteCon == nil || Node.RemoteCon.RemotePub == nil {
		return fmt.Errorf("Not connected to anyone\n")
	}

	var chat string
	for _, s := range args {
		chat += s + " "
	}
	msg := append([]byte{qln.MSGID_TEXTCHAT}, []byte(chat)...)

	_, err := Node.RemoteCon.Write(msg)
	return err
}

func Txs(args []string) error {
	alltx, err := SCon.TS.GetAllTxs()
	if err != nil {
		return err
	}
	for i, tx := range alltx {
		fmt.Printf("tx %d %s\n", i, uspv.TxToString(tx))
	}
	return nil
}

// Fake generates a fake tx and ingests it.  Needed in airplane mode.
// syntax is the same as send, but the inputs are invalid.
func Fake(args []string) error {

	// need args, fail
	if len(args) < 2 {
		return fmt.Errorf("need args: ssend address amount(satoshis) wit?")
	}
	adr, err := btcutil.DecodeAddress(args[0], SCon.TS.Param)
	if err != nil {
		fmt.Printf("error parsing %s as address\t", args[0])
		return err
	}

	amt, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return err
	}

	tx := wire.NewMsgTx() // make new tx
	// make address script 76a914...88ac or 0014...
	outAdrScript, err := txscript.PayToAddrScript(adr)
	if err != nil {
		return err
	}
	// make user specified txout and add to tx
	txout := wire.NewTxOut(amt, outAdrScript)
	tx.AddTxOut(txout)

	hash, err := chainhash.NewHashFromStr("23")
	if err != nil {
		return err
	}
	op := wire.NewOutPoint(hash, 25)
	txin := wire.NewTxIn(op, nil, nil)
	tx.AddTxIn(txin)

	_, err = SCon.TS.Ingest(tx, 0)
	if err != nil {
		return err
	}

	return nil
}

// Bal prints out your score.
func Bal(args []string) error {
	if SCon.TS == nil {
		return fmt.Errorf("Can't get balance, spv connection broken")
	}

	// new flag for which set of addresses to get info for
	if len(args) > 1 {
		peerIdx, err := strconv.ParseInt(args[0], 10, 32)
		if err != nil {
			return err
		}
		cIdx, err := strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			return err
		}

		qc, err := Node.GetQchanByIdx(uint32(peerIdx), uint32(cIdx))
		if err != nil {
			return err
		}
		return Node.QchanInfo(qc)
	}

	fmt.Printf(" ----- Account Balance ----- \n")
	fmt.Printf(" ----- Channels ----- \n")
	qcs, err := Node.GetAllQchans()
	if err != nil {
		return err
	}

	for _, q := range qcs {
		if q.CloseData.Closed {
			fmt.Printf("CLOSED ")

		} else {
			fmt.Printf("CHANNEL")
		}
		fmt.Printf(" %s h:%d (%d,%d) cap: %d\n",
			q.Op.Hash.String(), q.Height,
			q.KeyGen.Step[3]&0x7fffffff, q.KeyGen.Step[4]&0x7fffffff, q.Value)
	}
	fmt.Printf(" ----- utxos ----- \n")
	var allUtxos portxo.TxoSliceByAmt
	allUtxos, err = SCon.TS.GetAllUtxos()
	if err != nil {
		return err
	}
	// smallest and unconfirmed last (because it's reversed)
	sort.Sort(sort.Reverse(allUtxos))

	var score, confScore int64
	for i, u := range allUtxos {
		fmt.Printf("utxo %d %s h:%d a:%d\n",
			i, u.Op.String(), u.Height, u.Value)
		if u.Seq != 0 {
			fmt.Printf("seq:%d", u.Seq)
		}

		fmt.Printf("\t%s %s\n", u.Mode.String(), u.KeyGen.String())
		score += u.Value
		if u.Height != 0 {
			confScore += u.Value
		}
	}

	height, err := SCon.TS.GetDBSyncHeight()
	if err != nil {
		return err
	}
	atx, err := SCon.TS.GetAllTxs()
	if err != nil {
		return err
	}
	stxos, err := SCon.TS.GetAllStxos()
	if err != nil {
		return err
	}
	adrs, err := SCon.TS.GetAllAddresses()
	if err != nil {
		return err
	}

	for i, a := range adrs {

		oa, err := btcutil.NewAddressPubKeyHash(a.ScriptAddress(), SCon.Param)
		if err != nil {
			return err
		}

		fmt.Printf("address %d %s OR %s\n", i, oa.String(), a.String())

	}

	fmt.Printf("Total known txs: %d\n", len(atx))
	fmt.Printf("Known utxos: %d\tPreviously spent txos: %d\n",
		len(allUtxos), len(stxos))
	fmt.Printf("Total coin: %d confirmed: %d\n", score, confScore)
	fmt.Printf("DB sync height: %d\n", height)

	return nil
}

// Adr makes a new address.
func Adr(args []string) error {

	// if there's an arg, make 10 adrs
	if len(args) > 0 {
		if args[0] == "recover" {
			err := SCon.TS.RecoverAdrs()
			if err != nil {
				return err
			}
		} else {
			for i := 0; i < 10; i++ {
				_, err := SCon.TS.NewAdr160()
				if err != nil {
					return err
				}
			}
		}
	}

	if len(args) > 1 {
		for i := 0; i < 1000; i++ {
			_, err := SCon.TS.NewAdr160()
			if err != nil {
				return err
			}
		}
	}

	// always make one
	a160, err := SCon.TS.NewAdr160()
	if err != nil {
		return err
	}

	filt, err := SCon.TS.GimmeFilter()
	if err != nil {
		return err
	}

	SCon.Refilter(filt)

	wa, err := btcutil.NewAddressWitnessPubKeyHash(a160, SCon.Param)
	if err != nil {
		return err
	}
	fmt.Printf("made new address %s\n",
		wa.String())

	/*
		if args[0] == "recover" {
			tip, err := SCon.TS.GetDBSyncHeight()
			if err != nil {
				log.Fatal(err)
			}
			// copying hardcoded checkpoint from lit.go
			if tip == 0 {
				tip = 1036000
				err = SCon.TS.SetDBSyncHeight(tip)
				if err != nil {
					log.Fatal(err)
				}
			}
			err = SCon.AskForHeaders()
			if err != nil {
				log.Fatal(err)
			}
			err = rpcShellListen()
		}
	*/

	return nil
}

// Sweep sends every confirmed uxto in your wallet to an address.
// it does them all individually to there are a lot of txs generated.
// syntax: sweep adr
func Sweep(args []string) error {
	var err error
	var adr btcutil.Address
	if len(args) < 2 {
		return fmt.Errorf("sweep syntax: sweep adr howmany (drop)")
	}

	adr, err = btcutil.DecodeAddress(args[0], SCon.TS.Param)
	if err != nil {
		fmt.Printf("error parsing %s as address\t", args[0])
		return err
	}

	numTxs, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return err
	}
	if numTxs < 1 {
		return fmt.Errorf("can't send %d txs", numTxs)
	}

	var allUtxos portxo.TxoSliceByAmt
	allUtxos, err = SCon.TS.GetAllUtxos()
	if err != nil {
		return err
	}
	// smallest and unconfirmed last (because it's reversed)
	sort.Sort(sort.Reverse(allUtxos))

	if len(args) == 2 {
		for i, u := range allUtxos {
			if u.Height != 0 && u.Value > 10000 {
				tx, err := SCon.TS.SendOne(*allUtxos[i], adr)
				if err != nil {
					return err
				}
				err = SCon.NewOutgoingTx(tx)
				if err != nil {
					return err
				}
				numTxs--
				if numTxs == 0 {
					return nil
				}
			}
		}
		fmt.Printf("spent all confirmed utxos; not enough by %d\n", numTxs)
		return nil
	}
	// drop send temporarity out of order
	//	for i, u := range allUtxos {
	//		if u.Height != 0 {
	//			intx, outtx, err := SCon.TS.SendDrop(*allUtxos[i], adr)
	//			if err != nil {
	//				return err
	//			}
	//			err = SCon.NewOutgoingTx(intx)
	//			if err != nil {
	//				return err
	//			}
	//			err = SCon.NewOutgoingTx(outtx)
	//			if err != nil {
	//				return err
	//			}
	//			numTxs--
	//			if numTxs == 0 {
	//				return nil
	//			}
	//		}
	//	}

	return nil
}

// Fan generates a bunch of fanout.  Only for testing, can be expensive.
// syntax: fan adr numOutputs valOutputs witty
func Fan(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("fan syntax: fan adr numOutputs valOutputs")
	}
	adr, err := btcutil.DecodeAddress(args[0], SCon.TS.Param)
	if err != nil {
		fmt.Printf("error parsing %s as address\t", args[0])
		return err
	}
	numOutputs, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return err
	}
	valOutputs, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return err
	}

	adrs := make([]btcutil.Address, numOutputs)
	amts := make([]int64, numOutputs)

	for i := int64(0); i < numOutputs; i++ {
		adrs[i] = adr
		amts[i] = valOutputs + i
	}
	tx, err := SCon.TS.SendCoins(adrs, amts)
	if err != nil {
		return err
	}

	return SCon.NewOutgoingTx(tx)
}

// Send sends coins.
func Send(args []string) error {
	if SCon.RBytes == 0 {
		return fmt.Errorf("Can't send, spv connection broken")
	}
	// get all utxos from the database
	allUtxos, err := SCon.TS.GetAllUtxos()
	if err != nil {
		return err
	}
	var score int64 // score is the sum of all utxo amounts.  highest score wins.
	// add all the utxos up to get the score
	for _, u := range allUtxos {
		score += u.Value
	}

	// score is 0, cannot unlock 'send coins' acheivement
	if score == 0 {
		return fmt.Errorf("You don't have money.  Work hard.")
	}
	// need args, fail
	if len(args) < 2 {
		return fmt.Errorf("need args: ssend address amount(satoshis) wit?")
	}
	adr, err := btcutil.DecodeAddress(args[0], SCon.TS.Param)
	if err != nil {
		fmt.Printf("error parsing %s as address\t", args[0])
		return err
	}
	amt, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return err
	}
	if amt < 1000 {
		return fmt.Errorf("can't send %d, too small", amt)
	}

	fmt.Printf("send %d to address: %s \n",
		amt, adr.String())

	var adrs []btcutil.Address
	var amts []int64

	adrs = append(adrs, adr)
	amts = append(amts, amt)
	tx, err := SCon.TS.SendCoins(adrs, amts)
	if err != nil {
		return err
	}
	return SCon.NewOutgoingTx(tx)
}

// Msend mayyybe sends.
func MSend(args []string) error {
	if SCon.RBytes == 0 {
		return fmt.Errorf("Can't send, spv connection broken")
	}
	// get all utxos from the database
	allUtxos, err := SCon.TS.GetAllUtxos()
	if err != nil {
		return err
	}
	var score int64 // score is the sum of all utxo amounts.  highest score wins.
	// add all the utxos up to get the score
	for _, u := range allUtxos {
		score += u.Value
	}

	// score is 0, cannot unlock 'send coins' acheivement
	if score == 0 {
		return fmt.Errorf("You don't have money.  Work hard.")
	}
	// need args, fail
	if len(args) < 2 {
		return fmt.Errorf("need args: msend address amount(satoshis)")
	}
	adr, err := btcutil.DecodeAddress(args[0], SCon.TS.Param)
	if err != nil {
		fmt.Printf("error parsing %s as address\t", args[0])
		return err
	}
	amt, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return err
	}
	if amt < 1000 {
		return fmt.Errorf("can't send %d, too small", amt)
	}

	fmt.Printf("send %d to address: %s \n",
		amt, adr.String())

	// make address script 76a914...88ac or 0014...
	outAdrScript, err := txscript.PayToAddrScript(adr)
	if err != nil {
		return err
	}
	// make user specified txout and add to tx
	txout := wire.NewTxOut(amt, outAdrScript)
	txos := []*wire.TxOut{txout}

	ops, err := SCon.MaybeSend(txos)
	if err != nil {
		return err
	}

	fmt.Printf("got txid %s. Requested output is at index %d\n",
		ops[0].String(), ops[0].Index)
	return nil
}

// Rsend really sends
func RSend(args []string) error {
	if SCon.RBytes == 0 {
		return fmt.Errorf("Can't send, spv connection broken")
	}
	// need args, fail
	if len(args) < 1 {
		return fmt.Errorf("need args: rsend txid")
	}

	txid, err := chainhash.NewHashFromStr(args[0])
	if err != nil {
		return err
	}

	return SCon.ReallySend(txid)
}

// Nsend nah doesn't send
func NSend(args []string) error {
	if SCon.RBytes == 0 {
		return fmt.Errorf("Can't send, spv connection broken")
	}
	// need args, fail
	if len(args) < 1 {
		return fmt.Errorf("need args: nsend txid")
	}

	txid, err := chainhash.NewHashFromStr(args[0])
	if err != nil {
		return err
	}

	return SCon.NahDontSend(txid)
}

func Help(args []string) error {
	fmt.Printf("commands:\n")
	fmt.Printf("help adr bal send fake fan sweep lis con fund push cclose break exit\n")
	return nil
}
