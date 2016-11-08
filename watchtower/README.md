# watchtower - watch channels for fraudulent transactions

The watchtower package implements unlinkable channel monitoring and recovery for lightning network channels.

## Design

To most users dealing with channels, channels are identified by their outpoint (hash + 32 bit index), or by some stand in with a 1:1 mapping to outpoints.  Since the goal of this design is that the watchtower never learns the outpoint, channels must be identified another way: by penalty output address.

(Note that you could not identify channels at all, and simply have a mapping of commitment transactions to penalty transactions without grouping them.  This could work, but has several problems - it uses ~3-4X more data, and is very difficult to delete and recover old space.  Anonymity could be somewhat improved though, so it's could be worth looking into.)

The 20-byte penalty output pubkey hash identifies a watched channel.  There are 3 data stores associated with the channel: static, elkrem, and sigidx.  (Note that these aren't stored in the same place in the database)

### static

Data that stays the same for the duration of the channel.

DestPKH : the destination pub key hash of the penalty transaction.  Also the "name" of the channel.

HAKDBase : The HAKD base point which becomes the revocable pubkey in the commitment script.  This is the key which the watchtower receives signatures for.

TimeBase : The timeout base point, which becomes the timeout pubkey in the commitment script.  The watchtower never deals with signatures from this key, and only needs to know it to build the script hash pre-image.

Delay / fee : Delay should stay the same for the duration of the channel.  Dealing with changing fees is... TBD; it's static for now.

### elkrem

Stores the customer's elkrem receiver associated with the channel.  Overwritten each time, but never gets too big.

### sigidx

Stores signatures and partial txids.  This is where most of the data is.  This is stored in a separate database / tree which is sorted by txid.  The value associated with each txid is the signature, along with the commitment number so that the proper elkrem points can be generated.

## database

The database is structured based on the assumptions that fraudulent channel closes basically never happen.  But that transactions come in a lot, and
