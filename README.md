# lit - a lightning node you can run on your own
![Lit Logo](litlogo145.png)

[![Coverage Status](https://coveralls.io/repos/github/mit-dci/lit/badge.svg)](https://coveralls.io/github/mit-dci/lit)

Under development, not for use with real money.

## Setup

### Prerequisites
- [Git](https://git-scm.com/)

### Installing

1. Start by installing Go
 - [Go Installation](https://golang.org/doc/install)

2. Set your Go variables to match your installed paths are set correctly:
- `.../go/bin` (your install location) is in `$PATH` (Windows: Add the install location into your `PATH` System Variable)
- `$GOPATH` is set the location of where you want lit (and other projects) to be
-  optional: If you want to have packages download in a separate location than your installation add `$GOROOT` set to another location (Windows: Add )

3. Download the lit project with `go get github.com/mit-dci/lit`

### Building

1. Go to the location of your lit installation with your defined gopath variable (`$GOPATH` on Linux and `%GOPATH%` Windows) to the lit location
```
cd [gopath]/src/github.com/mit-dci/lit
```

2. If you try to build now with `go build` you will receive several errors such as
```
cannot find package "golang.org/x/crypto/nacl/secretbox"
...
cannot find package "golang.org/x/crypto/scrypt"
...
cannot find package [packageName]
...
```

3. In order to download all missing packages, do `go get ./...` or `go get .`

4. Go back to location of the lit folder if you are not already there ([Step 1](#building)) and try to rebuild the project.

5. You may now want to build `lit-af`, the text based client which controls the lit node using
```
cd cmd/lit-af
go build
```

6. To run lit use:
(Note : Windows users can take off ./ but may need to change lit to lit.exe in the second line.)
```
cd GOPATH/src/github.com/mit-dci/lit
./lit --tn3 true
```
The words `true`, `yes`, `1` can be used to specify that lit automatically connect to a set of populated seeds. It can also be replaced by the ip of the remote node you wish to connect to.

## Using Lightning

Great! Now that you are all done setting up lit, you can
- read about the arguments for starting lit [here](#command-line-arguments)
- read about the folders for the code and what does what [here](#folders)
- head over to the [Walkthrough](./WALKTHROUGH.md) to create some lit nodes or
- check out how to [Contribute](./CONTRIBUTING.md).



## Command line arguments
(a lit.conf file is not yet implemented but is on the TODO list)

When starting lit, the following command line arguments are available

#### connecting to networks:

| Arguments                   | Details                                                      | Default Port  |
| --------------------------- |--------------------------------------------------------------| ------------- |
| `--tn3 <nodeHostName>`      | connect to `nodeHostName`, which is a bitcoin testnet3 node. | 18333         |
| `--reg <nodeHostName>`      | connect to `nodeHostName`, which is a bitcoin regtest node.  | 18444         |
| `--lt4 <nodeHostName>`      | connect to `nodeHostName`, which is a litecoin testnet4 node.| 19335         |

#### other settings:

| Arguments                   | Details                                                      |
| --------------------------- |--------------------------------------------------------------|
| `-v` or `--verbose`         | Verbose; log everything to stdout as well as the lit.log file.  Lots of text.|
| `--dir <folderPath>`        | use `folderPath` as the directory.  By default, saves to `~/.lit/` |
| `-p` or `--rpcport <portNumber>` | listen for RPC clients on port `portNumber`.  Defaults to `8001`.  Useful when you want to run multiple lit nodes on the same computer (also need the `--dir` option) |
| `-r` or `--reSync`          | try to re-sync to the blockchain from the height given `-tip` |

## Folders

| Folder Name  | Details                                                                                                                                  |
|:-------------|:-----------------------------------------------------------------------------------------------------------------------------------------|
| `cmd`        | Has some rpc client code to interact with the lit node.  Not much there yet                                                              |
| `elkrem`     | A hash-tree for storing `log(n)` items instead of n                                                                                      |
| `litbamf`    | Lightning Network Browser Actuated Multi-Functionality -- web gui for lit                                                                |
| `litrpc`     | Websocket based RPC connection                                                                                                           |
| `lndc`       | Lightning network data connection -- send encrypted / authenticated messages between nodes                                               |
| `lnutil`     | Some widely used utility functions                                                                                                       |
| `portxo`     | Portable utxo format, exchangable between node and base wallet (or between wallets).  Should make this into a BIP once it's more stable. |
| `powless`    | Introduces a web API chainhook in addition to the uspv one                                                                               |
| `qln`        | A quick channel implementation with databases.  Doesn't do multihop yet.                                                                 |
| `sig64`      | Library to make signatures 64 bytes instead of 71 or 72 or something                                                                     |
| `test`       | Integration tests                                                                                                                        |
| `uspv`       | Deals with the network layer, sending network messages and filtering what to hand over to `wallit`                                       |
| `wallit`     | Deals with storing and retrieving utxos, creating and signing transactions                                                               |
| `watchtower` | Unlinkable outsourcing of channel monitoring                                                                                             |

### Hierarchy of packages

One instance of lit has one litNode (package qln).

LitNodes manage lndc connections to other litnodes, manage all channels, rpc listener, and the ln.db.  Litnodes then initialize and contol wallits.


A litNode can have multiple wallits; each must have different params.  For example, there can be a testnet3 wallit, and a regtest wallit.  Eventually it might make sense to support a root key per wallit, but right now the litNode gives a rootPrivkey to each wallet on startup.  Wallits each have a db file which tracks utxos, addresses, and outpoints to watch for the upper litNode.  Wallits do not directly do any network communication.  Instead, wallits have one or more chainhooks; a chainhook is an interface that talks to the blockchain.


One package that implements the chainhook interface is uspv.  Uspv deals with headers, wire messages to fullnodes, filters, and all the other mess that is contemporary SPV.

(in theory it shouldn't be too hard to write a package that implements the chainhook interface and talks to some block explorer.  Maybe if you ran your own explorer and authed and stuff that'd be OK.)


## License
[MIT](https://github.com/mit-dci/lit/blob/master/LICENSE)
