# lit - a lightning node you can run on your pwn
![Lit Logo](litlogo145.png)

Under development, not for use with real money.

## Installing on Linux 

1. Start by installing Go v1.6. 
 * Check this tutorial out: <https://www.digitalocean.com/community/tutorials/how-to-install-go-1-6-on-ubuntu-14-04>
  * TIP: You may want to install Go in a more localized place to keep your system cleaner from stray files.

2. Make sure your Go paths are set correctly: the `.../go/bin` path is in `$PATH` and `$GOROOT` is set to `.../go/`.

3. Create the directory to house the lit project (`mkdir .../lit`) and assign `$GOPATH` to that location.

3. Download the lit project: `go get github.com/mit-dci/lit`

4. The `go get` will fail. 
  * Lit uses btcd libraries, and btcd does not yet have segwit support in master.  This will hopefully be merged soon, before segwit activates on mainnet.
  * I have a fork of btcd from roasbeef/segwit, and could make it go-gettable by changing all the imports, but hopefully it will be merged in soon.
  * To make it work, switch to the adiabat/btcd libraries:  
```
# github.com/mit-dci/lit/uspv
uspv/eight333.go:406: undefined: wire.InvTypeWitnessBlock
... other errors.

user@host:~/go/src/github.com/mit-dci/lit$ cd ../../btcsuite/btcd/

user@host:~/go/src/github.com/btcsuite/btcd$ git remote add adiabat https://github.com/adiabat/btcd
user@host:~/go/src/github.com/btcsuite/btcd$ git fetch adiabat
user@host:~/go/src/github.com/btcsuite/btcd$ git checkout adiabat/master

user@host:~/go/src/github.com/btcsuite/btcd$ cd ../btcutil

user@host:~/go/src/github.com/btcsuite/btcutil$ git remote add adiabat https://github.com/adiabat/btcutil
user@host:~/go/src/github.com/btcsuite/btcutil$ git fetch adiabat
user@host:~/go/src/github.com/btcsuite/btcutil$ git checkout adiabat/master
```

5. Build, compile, and install: `go install`

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
