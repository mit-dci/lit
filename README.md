# lit - a lightning node you can run on your own

![Lit Logo](litlogo145.png)

[![Build Status](http://hubris.media.mit.edu:8080/job/lit-PR/badge/icon)](http://hubris.media.mit.edu:8080/job/lit-PR/)

Under development, not for use with real money.

## Setup

### Prerequisites

* [Git](https://git-scm.com/)

* [Go](https://golang.org/doc/install)

* make

* (Optional, Windows) a [Cygwin](https://cygwin.com/install.html) environment might make things easier for you.

### Downloading

Just use Git to download the repo and then cd into it.

```bash
git clone https://github.com/mit-dci/lit
cd lit
```

### Installation

#### Linux, macOS, Cygwin, etc.

Go has its own build process and dependency management, but for Lit it's a
little more complicated, so we set up a Makefile to do most of the work for you.

To build Lit and Lit-AF, just use the following command.  You don't have to have
a `$GOPATH` set up in any particular way since we handle that for you.

```bash
make # or `make all`
```

This will also run the test suite.  Running `make test with-python=true` will
include the python tests (requires `bitcoind`) that do some heavier testing.

#### Windows

It's recommended to install Cygwin and follow those setup instructions.  Or just
download prebuilt binaries.

1. Make sure that your `%GOPATH%` environmental variable is set up correctly.

2. Download dependencies and then build with something like this:

```
go get -v ./...
cd %GOPATH%\src\github.com\mit-dci\lit
go build -v .
go build -v .\cmd\lit-af
```

### Running

The below command will run Lit on the Bitcoin testnet3.  You probably need to
have `bitcoind` running on your machine such that Lit can connect to it when do
you this.

(Note: Windows users can take off `./` but may need to change `lit` to `lit.exe`
in the second line.)

```bash
./lit --tn3=true
```

Run `./lit --tn3=true` to start Lit on the Bitcoin testnet3

The words `yup, yes, y, true, 1, ok, enable, on` can be used to specify that Lit
automatically connect to a set of populated seeds. It can also be replaced by
the address of the remote node you wish to connect to.

### Packaging

This is a separate thing, but you can make a archive package for distribution by
using this:

```
./build/releasebuild.sh <os> <arch>
```

and it'll be dropped into `build/_releasedir`.  It should support any OS that
Go and our dependencies support.  Just instead of `windows` use `win` and
instead of `368` use `i386`.

**ALSO:** You can also package for Linux, macOS, and Windows in both amd64 and
i386 by just running `make package`.  (Except macOS is only amd64.)

And running `./build/releasebuild.sh clean` will cleanup the dirs it generates.

## Using Lightning

Great! Now that you are all done setting up lit, you can
- read about the arguments for starting lit [here](#command-line-arguments)
- read about the folders for the code and what does what [here](#folders)
- head over to the [Walkthrough](./WALKTHROUGH.md) to create some lit nodes or
- check out how to [Contribute](./CONTRIBUTING.md).

## Command line arguments

When starting lit, the following command line arguments are available.  The
following commands may also be specified in `lit.conf` which is automatically
generated on startup.

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
| `bech32`     | Util for the Bech32 spec                                                                                                                 |
| `btcutil`    | Misc bitcoin-specific libraries                                                                                                          |
| `build`      | Tools used for building Lit                                                                                                              |
| `cmd`        | Has some rpc client code to interact with the lit node.  Not much there yet                                                              |
| `coinparam`  | Information and other constants for identifying currencies                                                                               |
| `consts`     | Global constants                                                                                                                         |
| `crypto`     | Small utility cryptographic libraries                                                                                                    |
| `dlc`        | Discreet Log Contracts!                                                                                                                  |
| `docs`       | Other walkthroughs for doing things in Lit, also misc pictures                                                                           |
| `elkrem`     | A hash-tree for storing `log(n)` items instead of n                                                                                      |
| `litbamf`    | Lightning Network Browser Actuated Multi-Functionality -- web gui for lit                                                                |
| `litrpc`     | Websocket based RPC connection                                                                                                           |
| `lndc`       | Lightning network data connection -- send encrypted / authenticated messages between nodes                                               |
| `lnutil`     | Some widely used utility functions                                                                                                       |
| `portxo`     | Portable utxo format, exchangable between node and base wallet (or between wallets).  Should make this into a BIP once it's more stable. |
| `powless`    | Introduces a web API chainhook in addition to the uspv one                                                                               |
| `qln`        | A quick channel implementation with databases.  Doesn't do multihop yet.                                                                 |
| `sig64`      | Library to make signatures 64 bytes instead of 71 or 72 or something                                                                     |
| `snap`       | Snapcraft metadata                                                                                                                       |
| `test`       | Integration tests                                                                                                                        |
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

[MIT](https://github.com/mit-dci/lit/blob/master/LICENSE)
