# lit - a lightning node you can run on your pwn
![Lit Logo](litlogo145.png)

Under development, not for use with real money.

## Installing on Linux 

1. Start by installing Go 
 * Check this tutorial out: <https://www.digitalocean.com/community/tutorials/how-to-install-go-1-6-on-ubuntu-14-04>

2. Make sure your Go paths are set correctly: the `.../go/bin` path is in `$PATH`, `$GOROOT` is set to `.../go/`, and `$GOPATH` is the location of where you want lit to be. 

3. Download the lit project: `go get github.com/mit-dci/lit`

4. Run lit: 
```
user@host:~/go/src/github.com/mit-dci/lit$ ./lit -tn3 my.testnet3.node.tld
```

## Command line arguments
(a lit.conf file is not yet implemented but is on the TODO list)

When starting lit, the following command line arguments are available

#### connecting to networks:

-tn3 <nodeHostName>

connect to nodeHostName, which is a bitcoin testnet3 node.  Default port 18333

-reg <nodeHostName>

connect to <nodeHostName>, which is a bitcoin regtest node.  Default port 18444

-lt4 <nodeHostName>

connect to <nodeHostName>, which is a litecoin testnet4 node.  Default port 19335

#### other settings

-ez

use bloom filters unstead of downloading the whole block.  This fucntionality hasn't been maintained for a few months and may not work properly.

-v

Verbose; log everything to stdout as well as the lit.log file.  Lots of text.

-dir <folderPath>

use <folderPath> as the directory.  By default, saves to ~/.lit/

-rpcport <portNumber>

listen for RPC clients on port <portNumber>.  Defaults to 8001.  Useful when you want to run multiple lit nodes on the same computer (also need the -dir option)

-tip <height>

start synchronization of the blockchain from <height>.  (probably doesn't work right now)

-resync

try to re-sync to the blockchain from the height give in -tip


## Folders:

### cmd
has some rpc client code to interact with the lit node.  Not much there yet

### elkrem
a hash-tree for storing log(n) items instead of n

### lndc
lightning network data connection -- send encrypted / authenticated messages between nodes

### lnutil
some widely used utility functions

### portxo
portable utxo format, exchangable between node and base wallet (or between wallets).  Should make this into a BIP once it's more stable.

### qln
A quick, channel implementation with databases.  Doesn't do multihop yet.

### sig64
Library to make signatures 64 bytes instead of 71 or 72 or something

### watchtower
Unlinkable outsourcing of channel monitoring

### uspv
An spv wallet library


### Heirarchy of packages

One instance of lit has one litNode (package qln).

LitNodes manage lndc connections to other litnodes, manage all channels, rpc listener, and the ln.db.  Litnodes then initialize and contol wallits.

wallit

A litNode can have multiple wallits; each must have different params.  For example, there can be a testnet3 wallit, and a regtest wallit.  Eventually it might make sense to support a root key per wallit, but right now the litNode gives a rootPrivkey to each wallet on startup.  Wallits each have a db file which tracks utxos, addresses, and outpoints to watch for the upper litNode.  Wallits do not directly do any network communication.  Instead, wallits have one or more chainhooks; a chainhook is an interface that talks to the blockchain.

uspv

One package that implements the chainhook interface is uspv.  Uspv deals with headers, wire messages to fullnodes, filters, and all the other mess that is contemporary SPV.

(in theory it shouldn't be too hard to write a package that implements the chainhook interface and talks to some block explorer.  Maybe if you ran your own explorer and authed and stuff that'd be OK.)
