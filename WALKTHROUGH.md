## Lit 0.0 Walkthrough

2017-01-09 Lit sortof works on testnet.  There are known bugs / omissions / errors, and also unknown ones!  Reporting bugs / crashes is helpful, and fixing them is even more helpful.

This walkthrough is to set people up who want send payments over channels on a test network.

### Step 1: Files in place

Look at README.md to get set up with goalng, the dependencies and the code.  Build the binary, lit.  Also, co into cmd/lit-af and build the lit-af binary there.  lit-af is the text based client which controls the lit node.

If you have a full node, like bitcoind or btcd, start running that as well, either on testnet3 or regtest mode.

For this walkthrough, you will run 2 lit nodes and have them make channels.  It's more fun for the nodes to be on different computers controlled by different people, but you can run them on the same machine for testing.  You can also have the full node running on that machine as well.

In this example, there are 2 computers.  Most things should work the same with 1 computer; just make sure to have 2 different folders for the lit nodes.

Set up two folders:

Alice's setup

```
alice@pi2:~$ mkdir anode
alice@pi2:~$ cd gofolder/src/github.com/mit-dci/lit
alice@pi2:~/gofolder/src/github.com/mit-dci/lit$ go build
alice@pi2:~/gofolder/src/github.com/mit-dci/lit$ cp lit ~/anode/
alice@pi2:~/gofolder/src/github.com/mit-dci/lit$ cd cmd/lit-af
alice@pi2:~/gofolder/src/github.com/mit-dci/lit/cmd/lit-af$ go build
alice@pi2:~/gofolder/src/github.com/mit-dci/lit/cmd/lit-af$ cp lit-af ~/anode/
alice@pi2:~/gofolder/src/github.com/mit-dci/lit/cmd/lit-af$ cd ~/anode/
alice@pi2:~/anode$ 
```

Bob's setup
```
bob@pi3:~$ mkdir bnode
bob@pi3:~$ cd gofolder/src/github.com/mit-dci/lit
bob@pi3:~/gofolder/src/github.com/mit-dci/lit$ go build
bob@pi3:~/gofolder/src/github.com/mit-dci/lit$ cp lit ~/bnode/
bob@pi3:~/gofolder/src/github.com/mit-dci/lit$ cd cmd/lit-af
bob@pi3:~/gofolder/src/github.com/mit-dci/lit/cmd/lit-af$ go build
bob@pi3:~/gofolder/src/github.com/mit-dci/lit/cmd/lit-af$ cp lit-af ~/bnode/
bob@pi3:~/gofolder/src/github.com/mit-dci/lit/cmd/lit-af$ cd ~/bnode/
bob@pi3:~/bnode$ 
```

### Step 2: Run lit and sync up

Alice starts running lit and syncs up to the blockchain.  The lit node will print lots of stuff on the screen, but can't be controlled from here.

Alice connects to her full node, fullnode.net.  By default this is on testnet3, using port 18333.

```
alice@pi2:~/anode$ ./lit -spv fullnode.net
lit node v0.0
-h for list of options.
No file testkey.hex, generating.
passphrase: 
repeat passphrase: 

WARNING!! Key file not encrypted!!
```

Here Alice can type a passphrase to secure the wallet the newborn lit node is generating.  But she just pressed enter twice, because this is testnet.  Even though it's testnet, lit will show warnings about not using a passphrase.

Now in another window, alice connects to the lit node over RPC.

```
alice@pi2:~/anode$ ./lit-af 
lit-af# ls
entered command: ls

	Addresses:
0 H5qcg9CudN1KyHKx5h1R87tspxmZ4rPSypQ6w (muUy8o3VHrofZ5YjnyJjLkMufvckVyAWWs)
	Utxo: 0 Conf:0 Channel: 0
Sync Height 1060000

```

ls shows how much money Alice has, which is none.  She can get some from a faucet.

Bob can start a wallet in the same way; if bob's node is running on the same computer, he'll have to run lit with -rpcport to listen on a different port and -dir to store lit files in a different direcotry, and start lit-af with -p to connect to that port.  Once Alice and Bob are both set up, they can connect to each other.

### Step 3: Connect

Alice sets her node to listen:

```
lit-af# lis
entered command: lis

listening on :2448 with key n1ozwFWDbZXKYjqwySv3VaTxNZvdVmQcam
```

n1ozwFWDbZXKYjqwySv3VaTxNZvdVmQcam is Alice's node-ID (This format will change soon).  She's listening on port 2448, but any port can be specified with the `lis` command.

Bob can connect to Alice:

```
entered command: con n1ozwFWDbZXKYjqwySv3VaTxNZvdVmQcam@pi2

connected to peer 1
lit-af# say 1 Hi Alice!
entered command: say 2 Hi Alice!


lit-af# 
```

Bob puts the pubkey@hostname for Alice and connects.  Then he says hi to Alice.

### Step 4: Open a channel

Bob is connected to Alice and wants to open a payment channel.  Bob has already made sure he has channel-ready coins by sweeping some to himself using the sweep command to his segwit address.  If he has enough money and is connected to Alice, he can open a channel.

```
lit-af# fund 1 50000000 0
```

This opens a channel with peer 1 (Alice) with a channel capacity of 50,000,000 satoshis (half a coin), and sends 0 satoshis over in the creation process.  Bob starts out with all 50,000,000 satoshis in the channel, so only he can send to Alice.

### Step 5: Send micro-payments

(Do they count as micro-payments if testnet coins have zero value?)

use ls again to see that the channel is there, it will be labeled channel 1.

```
lit-af# push 1 200000 
```

This pushes 200,000 satoshis to the other side of the channel.  You can do this 200 trillion times before the channel needs to be closed.  (Actually, since I don't think you can send that many payments, the software will probably crash if you do manage to exceed 2^48)

### Step 6: Break the channel

You can close the channel cooperatively or non-cooperatively.  Uncooperative is maybe more fun.  Type:

```
lit-af# break 1
```

to non-cooperatively close the channel at the current state.  The node who breaks the channel has to wait 5 blocks before they can use their money.  The other node can spend immediately.  After a cooperative close, they can both spend imediately.  The `break` command does net need a connection to the other node.
