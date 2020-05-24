# Executing a Discreet Log Contract using LIT-AF
This tutorial explains how you can execute a Discreet Log Contract using the LIT command line client (LIT-AF). There's a separate tutorial on setting up two LIT nodes, you can find that [here](test-setup.md)

We'll be providing oracle keys and signatures you can use for testing, but if you want to use your own oracle, you can create one using our tutorial in either [Go](https://github.com/mit-dci/dlc-oracle-go/blob/master/TUTORIAL.md), [.NET Core](https://github.com/mit-dci/dlc-oracle-dotnet/blob/master/TUTORIAL.md) or [NodeJS](https://github.com/mit-dci/dlc-oracle-nodejs/blob/master/TUTORIAL.md)

## Step 1: Opening LIT-AF

In the tutorial for setting up nodes, you learnt how to connect to LIT using it's command line utility `lit-af`. Open two consoles and connect `lit-af` to both of the LIT nodes running on your machine.

## Step 2: Adding the oracle

Next, in both consoles execute the following command to add the Oracle to the nodes:

```
dlc oracle add <pubkey> <name>
```

The pubkey parameter is the public key of the oracle you'd like to use. If you're using your own oracle, from the tutorial linked above, it will print this out to the console. Otherwise, you can use the public key `03c0d496ef6656fe102a689abc162ceeae166832d826f8750c94d797c92eedd465`, like so:

```
dlc oracle add 03c0d496ef6656fe102a689abc162ceeae166832d826f8750c94d797c92eedd465 Tutorial
```

When successful, you'll see an output something like this:

```
Oracle successfully registered under ID 1
```

If the ID is different, because you've experimented with oracles before, make sure you substitute with the correct ID in the commands we use hereafter.

## Step 3: Creating the contract

Now that we have the oracle imported, we can create the contract. The contract is based on an expected publication in the future. For this purpose, the oracle publishes an R-point. This is a public key to the one-time signing key used by the oracle to sign this and only this publication. We need this when forming the contract.

If you're using your own oracle from the tutorial, it will print the R-point to the console as well. Otherwise, you can use the R-point `027168bba1aaecce0500509df2ff5e35a4f55a26a8af7ceacd346045eceb1786ad`.

If you want to learn more about the commands and their parameters, you can issue it with a `-h` parameter. For instance, in stead of `dlc contract setoracle 1 1` you can issue `dlc contract setoracle -h` and it will print out the usage and parameters for the command:

```
dlc contract setoracle <cid> <oid>
Configures a contract for using a specific oracle
cid The ID of the contract
oid The ID of the oracle
```

In order to create the contract we issue the following commands to `lit-af`. You only need to to this for one node, since we'll be sending the contract over to the other one. If this is not the first time creating a contract in LIT, the contract will get a new index. You would need to correct the first parameter to all the `dlc contract set...` commands in that case.

First, we create a new draft contract

```
dlc contract new
```

Multiple oracles supported only from RPC cals.
With lit-af utility you have to set oracles number to 1.

```
dlc contract setoraclesnumber 1 1
```

Then, we configure the contract to use the oracle we added in step 2. Remember, if this oracle got a different ID, you have to adjust the second parameter to this call.

```
dlc contract setoracle 1 1
```

Next, we configure the timestamp at which the contract will settle. This is a time in the past, since we already know the value and signature. It's just a hint to the system at this time.

```
dlc contract settime 1 1528848000
```

Now we have to configure refund timestamp after which the refund transaction becomes valid.

```
dlc contract setrefundtime 1 1528848000
```
In this case the refund transaction becomes valid at the same time.

Then, we configure the R-point for the contract, as mentioned earlier this is the public key to the one-time signing key used by the oracle to sign the value it will publish.

```
dlc contract setrpoint 1 027168bba1aaecce0500509df2ff5e35a4f55a26a8af7ceacd346045eceb1786ad
```

We configure the coin type to be Bitcoin Regtest:

```
dlc contract setcointype 1 257
```

We set the funding of the contract. Both peers will fund 1 BTC (100,000,000 satoshi)

```
dlc contract setfunding 1 100000000 100000000
```

Next, we determine the division in the contract. The oracle from the tutorial mentioned above publishes a value between 10000 and 20000. So, we can use that in the contract to determine that we get all the value in the contract when the published value is 20000, and our counter party will get all the value if it's 10000

```
dlc contract setdivision 1 20000 10000
```

Now the contract is fully configured. You can check this by issuing the `dlc contract view 1` command, which will print out something like this:

```
entered command: dlc contract view 1
Index                 : 1
Oracle public key     : [03c0...ee...d465]
Oracle R-point        : [0271...35...86ad]
Settlement time       : Wed Jun 13 00:00:00 UTC 2018
Funded by us          : 100000000
Funded by peer        : 100000000
Coin type             : 257
Peer                  : None
Status                : Draft

Payout division:

Oracle value         | Our payout           | Their payout        
------------------------------------------------------------------
                   0 |                    0 |            200000000
                3000 |                    0 |            200000000
                6000 |                    0 |            200000000
                9000 |                    0 |            200000000
               12000 |             40000000 |            160000000
               15000 |            100000000 |            100000000
               18000 |            160000000 |             40000000
               21000 |            200000000 |                    0
               24000 |            200000000 |                    0
               27000 |            200000000 |                    0
               30000 |            200000000 |                    0
```

## Step 4: Sending the contract to the other peer

Now that the contract is ready, we can send it to the other peer. We do this by issuing the `dlc contract offer` command, followed by the contract ID and the index of the peer we want to send it to.

```
dlc contract offer 1 1
```

When you go to the other LIT node, and issue `dlc contract ls` you will see a new contract appeared with index 1. You can view it by issuing `dlc contract view 1`

```
Index                 : 1
Oracle public key     : [03c0...ee...d465]
Oracle R-point        : [0271...35...86ad]
Settlement time       : Wed Jun 13 00:00:00 UTC 2018
Funded by us          : 100000000
Funded by peer        : 100000000
Coin type             : 257
Peer                  : Peer 1
Status                : Received offer, awaiting reply

Payout division:

Oracle value         | Our payout           | Their payout        
------------------------------------------------------------------
                   0 |            200000000 |                    0
                3000 |            200000000 |                    0
                6000 |            200000000 |                    0
                9000 |            200000000 |                    0
               12000 |            160000000 |             40000000
               15000 |            100000000 |            100000000
               18000 |             40000000 |            160000000
               21000 |                    0 |            200000000
               24000 |                    0 |            200000000
               27000 |                    0 |            200000000
               30000 |                    0 |            200000000
```

You might notice that the payout table is reversed to the one we saw on the first LIT node, which makes sense.

## Step 5: Accepting the contract

Now we can accept the contract from the other peer by issuing:

```
dlc contract accept 1
```

What will happen now in sequence, is:

* The nodes will exchange their funding inputs
* The nodes will exchange signatures for spending from the contract output
* The nodes will exchange signatures for the funding transaction
* Node 1 will publish the funding transaction

You should now generate one more block in the Bitcoin Core Debug console, so that the transaction is mined. Once you did that, issue the `dlc contract view 1`, and the status of the contract should show `Active` (on both peers)

```
entered command: dlc contract view 1
Index                 : 1
Oracle public key     : [03c0...ee...d465]
Oracle R-point        : [0271...35...86ad]
Settlement time       : Wed Jun 13 00:00:00 UTC 2018
Funded by us          : 100000000
Funded by peer        : 100000000
Coin type             : 257
Peer                  : Peer 1
Status                : Active
```

## Step 6: Settling the contract

Once the oracle publishes a value, we can settle the contract using the value and the oracle's signature. If you used your own oracle, the value and signature are printed to the console - if you didn't you can use the value `15161` and signature `9e349c50db6d07d5d8b12b7ada7f91d13af742653ff57ffb0b554170536faeac`

You can do the settlement on either of the two peers, but in this case we'll run it on the first peer (the one that offered the contract):

```
dlc contract settle 1 15161 9e349c50db6d07d5d8b12b7ada7f91d13af742653ff57ffb0b554170536faeac
```

What this does in order, is:
* The node you issued the command to will publish the settlement transaction corresponding to the value 15161 to the blockchain
* That node will also send a transaction claiming the output due him using the oracle signature and his own private key (combined) back to his wallet
* The other node will observe the transaction being published, and claim the output due him back to his wallet using just his private key

In order to trigger the other node to claim back his funds, you should generate two more blocks in the Bitcoin Core debug window.

## Step 7: Check balances

After mining the blocks, you will see that the balances and UTXOs on both peers have changed, if you issue the `ls` command:

Peer 1:
```
entered command: ls
[...]
	Txos:
0 2adb78b78ef287ec775acef77398040551a7ba362f708df3b8b200b40518b7ae;0 h:203 amt:103219000 /44'/257'/0'/0'/3' regtest
1 2fe0c7ec6fcb90911ba4f946421a6c73f660bd69ed2298018afa3b0ae3c200d0;1 h:203 amt:899999500 /44'/257'/0'/0'/2' regtest
[...]
	Type: 257	Sync Height: 204	FeeRate: 80	Utxo: 1003218500	WitConf: 1003218500 Channel: 0
```

Peer 2:
```
entered command: ls
[...]
	Txos:
0 7295f550fae9bb71689565251ee78dd22896decb490cdefa0ff5c6983563f6c8;0 h:203 amt:96779000 /44'/257'/0'/0'/2' regtest
1 2fe0c7ec6fcb90911ba4f946421a6c73f660bd69ed2298018afa3b0ae3c200d0;2 h:203 amt:899999500 /44'/257'/0'/0'/1' regtest
[...]
	Type: 257	Sync Height: 204	FeeRate: 80	Utxo: 996778500	WitConf: 996778500 Channel: 0
```

You can see that Peer 1 has 10.032185 BTC and peer 2 has 9.96778500 BTC. Both have an output of 8.999995 BTC, which is the change they got when funding the contract with 1 BTC (and paying 500 satoshi fees). The other output came from the contract based on the division. Since the published value was close to the middle of the contract (15000 would have equally divided the contract), the difference is not too big.

## Conclusion

We executed a discreet log contract using LIT's command line client. If you want to integrate this technology into your own application, or you have a use case that you think could leverage this technology - we also have an RPC client for LIT in [Go](https://github.com/mit-dci/lit-rpc-client-go), [.NET Core](https://github.com/mit-dci/lit-rpc-client-dotnet) and [NodeJS](https://github.com/mit-dci/lit-rpc-client-nodejs) that you can use to issue these commands programmatically. A tutorial on how to do that will follow.
