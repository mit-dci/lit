# Push / pull - flow for channel state updates

Here's how channels ("qChans" in the code, because you can't start anything with "chan" in go and nothing else started with q...) are updated.

There's one state at a time, with 2 variables which indicate aspects of a next or previous state: delta and prevRH.

States look like this

prev | cur | next
--- | --- | ---
 | idx |
 | amt | delta
prevRH | revH |
 | sig |

When nothing is inflight, delta is 0 and prevRev is an empty 20 byte array.

4 messages: RTS, ACKSIG, SIGREV, REV.
Pusher is the side which initiate the payment.  Payments are always "push" and requesting a "pull" is out of the scope of this protocol.  Puller is the one who recives funds.

Pusher - RTS: Request to send

Puller - ACKSIG: Acknowledge update, provide signature

Pusher - SIGREV: Sign new state, revoke old state

Puller - REV: Revoke old state

There's only 1 struct in ram so there's a bunch of overwrites.  But there's data on the disk in the DB, so if something fails, like signature verification, you restore from the DB.  It's safe in that you only ever have one state on the DB so you know what to broadcast.  You overwrite their sig, which is the dangerous part (don't want to keep track of sigs where you've revoked that state)

There's only one state in ram, and only one state on disk.  However, in terms of "previous / current / next", the state on disk may be earlier than the state in ram.  Ram is "ahead"; you're not sure from looking at the disk if you've sent the message or not, but you can just send again if you're not sure.  From looking at the state on disk, it is clear what the is next step and mesages to send.

State has a mutex in case something comes in over the wire before you're done modifying in-ram state.  Probably safer that way.

## Message and DB sequence:

### Pusher: UI trigger (destination, amountToSend)
load state from DB
RAM state: set delta to -amountToSend (delta is negative for pusher)
##### save to DB (only negative delta is new)
idx++
create theirHAKDPub(idx)
send RTS (amountToSend, theirHAKDPub)
release lock

### Puller: Receive RTS
load state from DB
check RTS(idx) == idx+1
check RTS(amount) > 0
delta = RTS(amount)
prevHAKD = myHAKDpub
myHAKDpub = RTS(HAKDpub)
##### Save to DB(positive delta and prevHAKD, HADKpub are new)
idx++
amt += delta
delta = 0
- create tx (theirs)
sign tx
create theirHAKDPub(idx)
send ACKSIG(sig, theirHAKDPub)
clear theirHAKDPub
release lock

### Pusher: Receive ACKSIG
load state from DB
idx++
amt += delta
delta = 0
create theirHAKDPub(idx)
sig = SIGACK(sig)
- create tx (mine)
verify sig (if fails, restore from DB, try RTS again..?)
clear theirHAKDPub (never saved to DB anyway...)
##### Save to DB(all fields new; prevHAKD populated, delta = 0)
prevHAKD = myHAKDpub
myHAKDpub = SIGACK(HAKDpub)
- create tx (theirs)
sign tx
create elk(idx-1)
send SIGREV(sig, elk)

### Puller: Receive SIGREV
load state from DB
idx++
amt += delta
delta = 0
sig = SIGREV(sig)
- create tx(mine)
verify sig (if fails, reload from DB, send ACKSIG again..? or record error?)
verify elk insertion (do this 2nd because can broadcast valid sig to close)
verify addPrivToBytes(priv, elk) == prevHAKD
##### Save to DB( sig fields new, prevRH empty, delta = 0)
create elk(idx-1)
send REV(elk)

### Pusher: Receive REV
verify elk insertion
verify addPrivToBytes(priv, elk) == myHAKDpub
set prevHAKD to empty
##### Save to DB(prevRH empty)

## Explanation

The genral sequence is to take in data, use it to modify the state in RAM, and  verify it.  If it's OK, then save it to the DB, then after saving construct and send the response.  This way if something goes wrong and you pull the plug, you might not be sure if you sent a message or not, but you can safely construct and send it again, based on the data in the DB.  Based on the DB state you'll know where in the process you stopped and can hopefully resume.

TLDR:

Pusher sends amount he's sending an a revokable pubkey.

puller makes pusher tx, signs and sends sig, revokable pubkey.

pusher makes pusher tx, verifies sig. makes puller tx, signs; sends sig and elkrem

puller makes puller tx, verifies sig. verifies pusher elkrem. sends elkrem.

pusher verifies puller elkrem.









