# RPC Commands

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
* `Data (32 byte array)``

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
