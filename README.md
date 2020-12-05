# lit - a lightning node you can run on your own

![Lit Logo](litlogo145.png)

[![Build Status](http://hubris.media.mit.edu:8080/job/lit-PR/badge/icon)](http://hubris.media.mit.edu:8080/job/lit/)

Under development, not for use with real money.

## Setup

### Prerequisites

* [Git](https://git-scm.com/)

* [Go](https://golang.org/doc/install)

* make

* (Optional, Windows) [Cygwin](https://cygwin.com/install.html)

* (Optional, for full test suite) Python 3 + `requests` library from PyPI

### Downloading

Clone the repo from git 

```bash
git clone https://github.com/mit-dci/lit
cd lit
```
or `go get` it
```go
go get -v github.com/mit-dci/lit
```

### Installation

#### Linux, macOS, Cygwin, etc.

You can either use Go's built-in dependency management and build tool
```go
cd {GOPATH}/src/github.com/mit-dci/lit
go get -v ./...
go build
```
or use the Makefile
```bash
make # or make all
```

To run the python integration tests (which requires `bitcoind`), run `make test with-python=false`

#### Windows

Install [Cygwin](http://www.cygwin.com) and follow the setup instructions or download prebuilt binaries from

1. Make sure that environmental variable `%GOPATH%` is initizlized correctly.

2. Download required dependencies and then build with:

```
go get -v ./...
cd %GOPATH%\src\github.com\mit-dci\lit
go build -v .
go build -v .\cmd\lit-af
```

### Running lit

The below command will run Lit on the Bitcoin testnet3 network

(Note: Windows users should take off `./` but need to change `lit` to `lit.exe`)

```bash
./lit --tn3=true
```

The words `yup, yes, y, true, 1, ok, enable, on` can be used to specify that Lit
automatically connect to peers fetched from a list of DNS seeds. It can also be replaced by
the address of the node you wish to connect to. For example for the btc testnet3:

```bash
./lit --tn3=localhost
```

It will use default port for different nodes. See the "Command line arguments" section.

### Packaging

You can make an archive package for any distribution by doing:

```
./build/releasebuild.sh <os> <arch>
```

and it will be placed in `build/_releasedir`.  It should support any OS that
Go and lit's dependencies support.  In place of `windows` use `win` and
in place of `386` use `i386`.

You can also package for Linux, macOS, and Windows in both amd64 and
i386 architectures by running `make package`. (NOTE: macOS is amd64 only)

Running `./build/releasebuild.sh clean` cleans the directories it generates.

## Using Lightning

Once you are done setting up lit, you can read about
- [the different command line arguments](#command-line-arguments)
- [the various folders](#folders) or
- [checkout the Walkthrough](./WALKTHROUGH.md)

## Contributing

Pull Requests and Issues are most welcome, checkout [Contributing](./CONTRIBUTING.md) to get started.

## Command line arguments

When starting lit, the following command line arguments are available.  The
following commands may also be specified in `lit.conf` which is automatically
generated on startup with `tn3=1` by default.

#### Connecting to networks

| Arguments                   | Details                                                      | Default Port  |
| --------------------------- |--------------------------------------------------------------| ------------- |
| `--tn3 <nodeHostName>`      | connect to `nodeHostName`, which is a bitcoin testnet3 node. | 18333         |
| `--reg <nodeHostName>`      | connect to `nodeHostName`, which is a bitcoin regtest node.  | 18444         |
| `--lt4 <nodeHostName>`      | connect to `nodeHostName`, which is a litecoin testnet4 node.| 19335         |

#### Other settings

| Arguments                        | Details                                                                                                                                                                |
| ---------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-v` or `--verbose`              | Verbose; log everything to stdout as well as the lit.log file.  Lots of text.                                                                                          |
| `--dir <folderPath>`             | Use `folderPath` as the directory.  By default, saves to `~/.lit/`.                                                                                                    |
| `-p` or `--rpcport <portNumber>` | Listen for RPC clients on port `portNumber`.  Defaults to `8001`.  Useful when you want to run multiple lit nodes on the same computer (also need the `--dir` option). |
| `-r` or `--reSync`               | Try to re-sync to the blockchain.                                                                                                                                      |

## Folders

| Folder Name  | Details                                                                                                                                  |
|:-------------|:-----------------------------------------------------------------------------------------------------------------------------------------|
| `bech32`     | Util for the Bech32 format                                                                                                             |
| `btcutil`    | Bitcoin-specific libraries                                                                                                          |
| `build`      | Tools used for building Lit                                                                                                              |
| `cmd`        | Has some rpc client code to interact with the lit node.  Not much there yet                                                              |
| `coinparam`  | Information and other constants for identifying currencies                                                                               |
| `consts`     | Global constants                                                                                                                         |
| `crypto`     | Utility cryptographic libraries                                                                                                    |
| `dlc`        | Discreet Log Contracts                                                                                                                   |
| `docs`       | Writeups for setting up things and screenshots                 |
| `elkrem`     | A hash-tree for storing `log(n)` items instead of n                                                                                      |
| `litrpc`     | Websockets based RPC connection                                                                                                           |
| `lndc`       | Lightning network data connection -- send encrypted / authenticated messages between nodes                                               |
| `lnutil`     | Widely used utility functions                                                                                                       |
| `portxo`     | Portable utxo format, exchangable between node and base wallet (or between wallets).  Should make this into a BIP once it's more stable. |
| `powless`    | Introduces a web API chainhook in addition to the uspv one                                                                               |
| `qln`        | A quick channel implementation with databases.  Doesn't do multihop yet.                                                                 |
| `sig64`      | Library to make signatures 64 bytes instead of 71 or 72 or something                                                                     |
| `snap`       | Snapcraft metadata                                                                                                                       |
| `test`       | Python Integration tests                                                                                                                        |
| `uspv`       | Deals with the network layer, sending network messages and filtering what to hand over to `wallit`                                       |
| `wallit`     | Deals with storing and retrieving utxos, creating and signing transactions                                                               |
| `watchtower` | Unlinkable outsourcing of channel monitoring                                                                                             |
| `wire`       | Tools for working with binary data structures in Bitcoin                                                                                 |

### Hierarchy of packages

One instance of lit has one litNode (package qln).

LitNodes manage lndc connections to other litnodes, manage all channels, rpc listener, and the ln.db.  Litnodes then initialize and contol wallits.

A litNode can have multiple wallits; each must have different params.  For example, there can be a testnet3 wallit, and a regtest wallit.  Eventually it might make sense to support a root key per wallit, but right now the litNode gives a rootPrivkey to each wallet on startup.  Wallits each have a db file which tracks utxos, addresses, and outpoints to watch for the upper litNode.  Wallits do not directly do any network communication.  Instead, wallits have one or more chainhooks; a chainhook is an interface that talks to the blockchain.

One package that implements the chainhook interface is uspv.  Uspv deals with headers, wire messages to fullnodes, filters, and all the other mess that is contemporary SPV.

(in theory it shouldn't be too hard to write a package that implements the chainhook interface and talks to some block explorer.  Maybe if you ran your own explorer and authed and stuff that'd be OK.)

#### Dependency graph

![Dependency Graph](docs/deps.png)

## License

Some modules imported from other libraries have their own LICENSE files attached. For all other files, this repository uses a [MIT](https://github.com/mit-dci/lit/blob/master/LICENSE) license.
