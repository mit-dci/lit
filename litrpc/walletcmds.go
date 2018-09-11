package litrpc

import (
	"fmt"

	"github.com/mit-dci/lit/bech32"
	"github.com/mit-dci/lit/consts"
	"github.com/mit-dci/lit/lnutil"
	"github.com/mit-dci/lit/logging"
	"github.com/mit-dci/lit/portxo"
	"github.com/mit-dci/lit/wire"
)

type TxidsReply struct {
	Txids []string
}
type StatusReply struct {
	Status string
}

type NoArgs struct {
	// nothin
}

type CoinArgs struct {
	CoinType uint32
}

// ------------------------- balance
// BalReply is the reply when the user asks about their balance.
type CoinBalReply struct {
	CoinType    uint32
	SyncHeight  int32 // height this wallet is synced to
	ChanTotal   int64 // total balance in channels
	TxoTotal    int64 // all utxos
	MatureWitty int64 // confirmed, spendable and witness
	FeeRate     int64 // fee per byte
}

type BalanceReply struct {
	Balances []CoinBalReply
}

func (r *LitRPC) Balance(args *NoArgs, reply *BalanceReply) error {

	var allTxos portxo.TxoSliceByAmt

	// get all channels
	qcs, err := r.Node.GetAllQchans()
	if err != nil {
		return err
	}

	for cointype, wal := range r.Node.SubWallet {
		// will add the balance for this wallet to the full reply
		var cbr CoinBalReply

		cbr.CoinType = cointype
		// get wallet height
		cbr.SyncHeight = wal.CurrentHeight()
		// also current fee rate
		cbr.FeeRate = wal.Fee()

		allTxos, err = wal.UtxoDump()
		if err != nil {
			return err
		}

		// ask sub-wallet for balance
		cbr.TxoTotal = allTxos.Sum()
		cbr.MatureWitty = allTxos.SumWitness(cbr.SyncHeight)

		// iterate through channels to figure out how much we have
		for _, q := range qcs {
			if q.Coin() == cointype && !q.CloseData.Closed {
				cbr.ChanTotal += q.State.MyAmt
			}
		}

		// I thought slices were pointery enough that I could put this line
		// near the top.  Guess not.
		reply.Balances = append(reply.Balances, cbr)
	}
	return nil
}

type TxoInfo struct {
	OutPoint string
	Amt      int64
	Height   int32
	Delay    int32
	CoinType string
	Witty    bool

	KeyPath string
}
type TxoListReply struct {
	Txos []TxoInfo
}

// TxoList sends back a list of all non-channel utxos
func (r *LitRPC) TxoList(args *NoArgs, reply *TxoListReply) error {

	for _, wal := range r.Node.SubWallet {

		walTxos, err := wal.UtxoDump()
		if err != nil {
			return err
		}

		syncHeight := wal.CurrentHeight()

		theseTxos := make([]TxoInfo, len(walTxos))
		for i, u := range walTxos {
			theseTxos[i].OutPoint = u.Op.String()
			theseTxos[i].Amt = u.Value
			theseTxos[i].Height = u.Height
			theseTxos[i].CoinType = wal.Params().Name
			// show delay before utxo can be spent
			if u.Seq != 0 {
				theseTxos[i].Delay = u.Height + int32(u.Seq) - syncHeight
			}
			theseTxos[i].Witty = u.Mode&portxo.FlagTxoWitness != 0
			theseTxos[i].KeyPath = u.KeyGen.String()
		}

		reply.Txos = append(reply.Txos, theseTxos...)
	}
	return nil
}

// ------------------------- send
type SendArgs struct {
	DestAddrs []string
	Amts      []int64
}

func (r *LitRPC) Send(args SendArgs, reply *TxidsReply) error {
	var err error

	nOutputs := len(args.DestAddrs)
	if nOutputs < 1 {
		return fmt.Errorf("No destination address specified")
	}
	if nOutputs != len(args.Amts) {
		return fmt.Errorf("%d addresses but %d amounts specified",
			nOutputs, len(args.Amts))
	}
	// get cointype for first address.
	coinType := CoinTypeFromAdr(args.DestAddrs[0])
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[coinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for address %s type %d",
			args.DestAddrs[0], coinType)
	}
	// All addresses must have the same cointype as they all
	// must to be in the same tx.
	for _, a := range args.DestAddrs {
		if CoinTypeFromAdr(a) != coinType {
			return fmt.Errorf("Coin type mismatch for address %s, %s",
				a, args.DestAddrs[0])
		}
	}

	txOuts := make([]*wire.TxOut, nOutputs)
	for i, s := range args.DestAddrs {
		if args.Amts[i] < consts.MinSendAmt {
			return fmt.Errorf("Amt %d less than minimum send amount %d", args.Amts[i], consts.MinSendAmt)
		}

		outScript, err := AdrStringToOutscript(s)
		if err != nil {
			return err
		}

		txOuts[i] = wire.NewTxOut(args.Amts[i], outScript)
	}

	// we don't care if it's witness or not
	ops, err := wal.MaybeSend(txOuts, false)
	if err != nil {
		return err
	}

	err = wal.ReallySend(&ops[0].Hash)
	if err != nil {
		return err
	}

	reply.Txids = append(reply.Txids, ops[0].Hash.String())
	return nil
}

// ------------------------- sweep
type SweepArgs struct {
	DestAdr string
	NumTx   uint32
	Drop    bool
}

func (r *LitRPC) Sweep(args SweepArgs, reply *TxidsReply) error {
	// get cointype for first address.
	coinType := CoinTypeFromAdr(args.DestAdr)
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[coinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for address %s type %d",
			args.DestAdr, coinType)
	}

	outScript, err := AdrStringToOutscript(args.DestAdr)
	if err != nil {
		return err
	}

	logging.Infof("numtx: %d\n", args.NumTx)
	if args.NumTx < 1 {
		return fmt.Errorf("can't send %d txs", args.NumTx)
	}

	txids, err := wal.Sweep(outScript, args.NumTx)
	if err != nil {
		return err
	}

	for _, txid := range txids {
		reply.Txids = append(reply.Txids, txid.String())
	}

	return nil
}

// ------------------------- fanout
type FanArgs struct {
	DestAdr      string
	NumOutputs   uint32
	AmtPerOutput int64
}

func (r *LitRPC) Fanout(args FanArgs, reply *TxidsReply) error {
	if args.NumOutputs < 1 {
		return fmt.Errorf("Must have at least 1 output")
	}
	if args.AmtPerOutput < 5000 {
		return fmt.Errorf("Minimum 5000 per output")
	}

	// get cointype for first address.
	coinType := CoinTypeFromAdr(args.DestAdr)
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[coinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for address %s type %d",
			args.DestAdr, coinType)
	}

	outScript, err := AdrStringToOutscript(args.DestAdr)
	if err != nil {
		return err
	}

	txos := make([]*wire.TxOut, args.NumOutputs)

	for i, _ := range txos {
		txos[i] = new(wire.TxOut)
		txos[i].Value = args.AmtPerOutput + int64(i)
		txos[i].PkScript = outScript
	}

	// don't care if inputs are witty or not
	ops, err := wal.MaybeSend(txos, false)
	if err != nil {
		return err
	}
	err = wal.ReallySend(&ops[0].Hash)
	if err != nil {
		return err
	}

	reply.Txids = append(reply.Txids, ops[0].String())
	return nil
}

// set fee
type SetFeeArgs struct {
	Fee      int64
	CoinType uint32
}

// get fee
type FeeArgs struct {
	CoinType uint32
}
type FeeReply struct {
	CurrentFee int64
}

// SetFee allows you to set a fee rate for a wallet.
func (r *LitRPC) SetFee(args *SetFeeArgs, reply *FeeReply) error {
	// if cointype is 0, use the node's default coin
	if args.CoinType == 0 {
		args.CoinType = r.Node.DefaultCoin
	}
	if args.Fee < 0 {
		return fmt.Errorf("Invalid value for SetFee: %d", args.Fee)
	}
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[args.CoinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for coin type %d", args.CoinType)
	}
	reply.CurrentFee = wal.SetFee(args.Fee)
	return nil
}

// Fee gets the fee rate for a wallet.
func (r *LitRPC) GetFee(args *FeeArgs, reply *FeeReply) error {
	// if cointype is 0, use the node's default coin
	if args.CoinType == 0 {
		args.CoinType = r.Node.DefaultCoin
	}
	// make sure we support that coin type
	wal, ok := r.Node.SubWallet[args.CoinType]
	if !ok {
		return fmt.Errorf("no connnected wallet for coin type %d", args.CoinType)
	}
	reply.CurrentFee = wal.Fee()
	return nil
}

// ------------------------- address
type AddressArgs struct {
	NumToMake uint32
	CoinType  uint32
}

// TODO Make this contain an array of structures not a structure of arrays.
type AddressReply struct {
	CoinTypes       []uint32
	WitAddresses    []string
	LegacyAddresses []string
}

func (r *LitRPC) Address(args *AddressArgs, reply *AddressReply) error {
	var allAdr [][20]byte
	var ctypesPerAdr []uint32

	// if cointype is 0, use the node's default coin
	if args.CoinType == 0 {
		args.CoinType = r.Node.DefaultCoin
	}

	// If you tell it to make 0 new addresses, it sends a list of all the old ones
	// (from every wallet)
	if args.NumToMake == 0 {
		// this gets 20 byte addresses; need to convert them to bech32 / base58
		// iterate through every wallet
		for cointype, wal := range r.Node.SubWallet {
			walAdr, err := wal.AdrDump()
			if err != nil {
				return err
			}

			for _, _ = range walAdr {
				ctypesPerAdr = append(ctypesPerAdr, cointype)
			}
			allAdr = append(allAdr, walAdr...)
		}
	} else {
		// if you have non-zero NumToMake, then cointype matters
		wal, ok := r.Node.SubWallet[args.CoinType]
		if !ok {
			return fmt.Errorf("No wallet of cointype %d linked", args.CoinType)
		}

		// call NewAdr a bunch of times
		remaining := args.NumToMake
		for remaining > 0 {
			adr, err := wal.NewAdr()
			if err != nil {
				return err
			}
			allAdr = append(allAdr, adr)
			ctypesPerAdr = append(ctypesPerAdr, args.CoinType)
			remaining--
		}
	}

	reply.CoinTypes = make([]uint32, len(allAdr))
	reply.WitAddresses = make([]string, len(allAdr))
	reply.LegacyAddresses = make([]string, len(allAdr))

	for i, a := range allAdr {

		// Store the cointype
		reply.CoinTypes[i] = ctypesPerAdr[i]

		// convert 20 byte array to old address
		param := r.Node.SubWallet[ctypesPerAdr[i]].Params()

		oldadr := lnutil.OldAddressFromPKH(a, param.PubKeyHashAddrID)
		reply.LegacyAddresses[i] = oldadr

		// convert 20-byte PKH to a bech32 segwit v0 address
		bech32adr, err := bech32.SegWitV0Encode(param.Bech32Prefix, a[:])

		if err != nil {
			return err
		}
		reply.WitAddresses[i] = bech32adr
	}

	return nil
}

// More human-readable replies
func (r *LitRPC) GetAddresses(args *NoArgs, reply *AddressReply) error {

	// return index
	ri := 0

	cts := make([]uint32, 0)
	was := make([]string, 0)
	las := make([]string, 0)

	for cointype, wal := range r.Node.SubWallet {

		walAdr, err := wal.AdrDump()
		if err != nil {
			panic("this should never happen, I don't think")
		}

		for _, pubkey := range walAdr {

			param := r.Node.SubWallet[cointype].Params()
			cts = append(cts, cointype)

			b32, err := bech32.SegWitV0Encode(param.Bech32Prefix, pubkey[:])
			if err != nil {
				panic("error encoding bech32 address")
			}

			was = append(was, b32)
			las = append(las, lnutil.OldAddressFromPKH(pubkey, param.PubKeyHashAddrID))

			ri++
		}

	}

	reply.CoinTypes = cts
	reply.WitAddresses = was
	reply.LegacyAddresses = las

	return nil
}

//func oldAddressPubKeyHash(pkHash []byte, netID byte) (string, error) {
//	// Check for a valid pubkey hash length.
//	if len(pkHash) != ripemd160.Size {
//		return "", errors.New("pkHash must be 20 bytes")
//	}
//	return base58.CheckEncode(pkHash, netID), nil
//}

type ClaimHTLCArgs struct {
	R [16]byte
}

func (r *LitRPC) ClaimHTLC(args *ClaimHTLCArgs, reply *TxidsReply) error {
	txids, err := r.Node.ClaimHTLC(args.R)
	if err != nil {
		return err
	}

	reply.Txids = make([]string, 0)
	for _, txid := range txids {
		reply.Txids = append(reply.Txids, fmt.Sprintf("%x", txid))
	}

	return nil
}
