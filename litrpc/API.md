# RPC Commands

Other structs referenced in request/responses have their declarations copied in
second secton of this document.

## chancmds

### ChannelList

Args:

* `ChanIdx (uint32)`

Returns:

* `Channels (ChannelInfo list)`

If ChanIdx is nonzero, returns info for only that channel.

### FundChannel

Args:

* `Peer (uint32)`
* `CoinType (uint32)`
* `Capacity (int64)`
* `Roundup (int64)`
* `InitialSend (int64)`
* `Data (32 byte array)`

Returns:

* `Status (string)`

### DualFundChannel

Args:

* `Peer (uint32)`
* `CoinType (uint32)`
* `OurAmount (int64)`
* `TheirAmount (int64)`

Returns:

* `Status (string)`

### DualFundDecline

Args: *empty object*

Returns:

* `Status (string)`

### DualFundAccept

Args: *empty object*

Returns:

* `Status (string)`

### PendingDualFund

Args: *empty object*

Returns:

* `Pending (bool)`
* `PeerIdx (uint32)`
* `CoinType (uint32)`
* `TheirAmount (int64)`
* `RequestedAmount (int64)`

### StateDump

Args: *empty object*

Returns:

* `Txs (JusticeTx list)`

### Push

Args:

* `ChanIdx (uint32)`
* `Amt (int64)`
* `Data (32 byte array)`

Returns:

* `StateIndex (uint64)``

### CloseChannel

Args:

* `ChanIdx (uint32)`

Returns:

* `Status (string)`

### BreakChannel

Args:

* `ChanIdx (uint32)`

Returns:

* `Status (string)`

### DumpPrivs

Args: *none*

Returns:

* `Privs (PrivInfo list)`

## dlccmds

### ListOracles

Args: *empty object*

Returns:

* `Oracles (DlcOracle list)`

### ImportOracle

Args:

* `Url (string)`
* `Name (string)`

Returns:

* `Oracle (DlcOracle)`

### AddOracle

Args:

* `Key (string)`
* `Name (string)`

Returns:

* `Oracle (DlcOracle)`

### NewContract

Args: *empty object*

Returns:

* `Contract (DlcContract)`

### ListContracts

Args: *empty object*

Returns:

* `Contracts (DlcContract list)`

### GetContract

Args:

* `Idx (uint64)`

Returns:

* `Contract (DlcContract)`

### SetContractOracle

Args:

* `CIdx (uint64)`
* `OIdx (uint64)`

Returns:

* `Success (bool)`

### SetContractDatafeed

Args:

* `CIdx (uint64)`
* `Feed (uint64)`

Returns:

* `Success (bool)`

### SetContractRPoint

Args:

* `CIdx (uint64)`
* `RPoint (33 byte list)`

Returns:

* `Success (bool)`

### SetContractFunding

Args:

* `CIdx (uint64)`
* `OutAmount (int64)`
* `TheirAmount (int64)`

### SetContractDivision

Args:

* `CIdx (uint64)`
* `ValueFullyOurs (int64)`
* `ValueFullyTheirs (int64)`

Returns:

* `Success (bool)`

### SetContractCoinType

Args:

* `CIdx (uint64)`
* `CoinType (uint32)`

Returns:

* `Success (bool)`

### OfferContract

Args:

* `CIdx (uint64)`
* `PeerIdx (uint32)`

Returns:

* `Success (bool)`

### DeclineContract

Args:

* `CIdx (uint64)`

Returns:

* `Success (bool)`

### AcceptContract

Args:

* `CIdx (uint64)`

Returns:

* `Success (bool)`

### SettleContract

Args:

* `CIdx (uint64)`
* `OracleValue (int64)`
* `OracleSig (32 byte list)`

Returns:

* `Success (bool)`
* `SettleTxHash (32 byte list)`
* `ClaimTxHash (32 byte list)`

## netcmds

### Listen

Args:

* `Port (string)` (why is this a string lol)

Returns:

* `LisIpPorts (string list)`
* `Adr (string)`

### Connect

Args:

* `LNAddr (string)`

Returns:

* `Status (string)`

### AssignNickname

Args:

* `Peer (uint32)`
* `Nickname (string)`

Returns:

* `Status (string)`

### ListConnections

Args: *none*

Returns:

* `Connections (PeerInfo list)`
* `MyPKH (string)`

### GetListeningPorts

Args: *none*

Returns:

* `LisIpPorts (string list)`
* `Adr (string)`

### GetMessages

Args: *none*

Returns:

* `Status (string)`

### Say

Args:

* `Peer (uint32)`
* `Message (string)`

Returns:

* `Status (string)`

### Stop

Args: *none*

Returns:

* `Status (string)`

### GetChannelMap

Args: *none*

Returns:

* `Graph (string)`

The string is formatted in the GraphViz `.dot` format.

## towercmds

### Watch

Args:

* `ChanIdx (uint32)`
* `SendToPeer (uint32)`

Returns:

* `Msg (string)`

## walletcmds

### Balance

Args: *none*

Returns:

* `Balances (CoinBalReply list)`

### TxoList

Args: *none*

Returns:

* `Txos (TxoInfo list)`

### Send

Args:

* `DestArgs (string list)`
* `Amts (int64 list)`

Returns:

* `Txids (string list)`

### Sweep

Args:

* `DestAdr (string)`
* `NumTx (uint32)`
* `Drop (bool)`

Returns:

* `Txids (string list)`

### Fanout

Args:

* `DestAdr (string)`
* `NumOutputs (uint32)`
* `AmtPerOutput (int64)`

Returns:

* `Txids (string list)`

### SetFee

Args:

* `Fee (int64)`
* `CoinType (uint32)`

Returns:

* `CurrentFee (int64)`

### GetFee

Args:

* `CoinType (uint32)`

Returns:

* `CurrentFee (int64)`

### Address

Args:

* `NumToMake (uint32)`
* `CoinType (uint32)`

Returns:

* `WitAddresses (string list)`
* `LegacyAddresses (string list)`

# Other Types

### ChannelInfo

```go
type ChannelInfo struct {
	OutPoint      string
	CoinType      uint32
	Closed        bool
	Capacity      int64
	MyBalance     int64
	Height        int32  // block height of channel fund confirmation
	StateNum      uint64 // Most recent commit number
	PeerIdx, CIdx uint32
	PeerID        string
	Data          [32]byte
	Pkh           [20]byte
}
```

### Priv Info

```go
type PrivInfo struct {
	OutPoint string
	Amt      int64
	Height   int32
	Delay    int32
	CoinType string
	Witty    bool
	PairKey  string

	WIF string
}
```

### DlcOracle

```go
type DlcOracle struct {
	Idx  uint64   // Index of the oracle for refencing in commands
	A    [33]byte // public key of the oracle
	Name string   // Name of the oracle for display purposes
	Url  string   // Base URL of the oracle, if its REST based (optional)
}
```

### DlcContract

```go
type DlcContract struct {
	// Index of the contract for referencing in commands
	Idx uint64
	// Index of the contract on the other peer (so we can reference it in
	// messages)
	TheirIdx uint64
	// Index of the peer we've offered the contract to or received the contract
	// from
	PeerIdx uint32
	// Coin type
	CoinType uint32
	// Pub keys of the oracle and the R point used in the contract
	OracleA, OracleR [33]byte
	// The time we expect the oracle to publish
	OracleTimestamp uint64
	// The payout specification
	Division []DlcContractDivision
	// The amounts either side are funding
	OurFundingAmount, TheirFundingAmount int64
	// PKH to which the contracts funding change should go
	OurChangePKH, TheirChangePKH [20]byte
	// Pubkey used in the funding multisig output
	OurFundMultisigPub, TheirFundMultisigPub [33]byte
	// Pubkey to be used in the commit script (combined with oracle pubkey
	// or CSV timeout)
	OurPayoutBase, TheirPayoutBase [33]byte
	// Pubkeyhash to which the contract pays out (directly)
	OurPayoutPKH, TheirPayoutPKH [20]byte
	// Status of the contract
	Status DlcContractStatus
	// Outpoints used to fund the contract
	OurFundingInputs, TheirFundingInputs []DlcContractFundingInput
	// Signatures for the settlement transactions
	TheirSettlementSignatures []DlcContractSettlementSignature
	// The outpoint of the funding TX we want to spend in the settlement
	// for easier monitoring
	FundingOutpoint wire.OutPoint
}
```

### PeerInfo

```go
type PeerInfo struct {
	PeerNumber uint32
	RemoteHost string
	LitAdr 	   string
	Nickname   string
}
```

### CoinBalReply

```go
type CoinBalReply struct {
	CoinType    uint32
	SyncHeight  int32 // height this wallet is synced to
	ChanTotal   int64 // total balance in channels
	TxoTotal    int64 // all utxos
	MatureWitty int64 // confirmed, spendable and witness
	FeeRate     int64 // fee per byte
}
```

### TxoInfo

```go
type TxoInfo struct {
	OutPoint string
	Amt      int64
	Height   int32
	Delay    int32
	CoinType string
	Witty    bool

	KeyPath string
}
```
