# lit - a lightning node you can run on your pwn

Under development, not for use with real money.

folders:

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
