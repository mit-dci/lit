# lit invoices

litvoice?  nah

prompt?  Prompt sounds cool.  It means fast.  Cool name for a startup.  The logo could be a $, which is like money, or #, which is root.  Nerdy, because that's a command prompt.  Also, even nerdier, it means prompt critical, kaboom.

litprompt?  I dunno, something like that.

So with lightning the receiver is online and there's interaction.  That's a little bit of a downside compared to on-chain non-interactive, but that has lots of issues anyway (address re-use).  Since the assumption is there's an authenticated data connection between the sender and receiver of funds, we should exploit that and minimize the control data that's sent out of band, and maximize control data sent in-band.

The BOLT11 spec does the opposite of this, by encoding all the data into a single blob which users have to deal with.  With work-shortened identifiers, we can instead have a payement request that looks like this:

```ln1d6qejxtdg4y5r3zarva:z```

Lets call this a payment prompt (for now)

The prompt does not encode all the payment details.  Instead it encodes just enough to authenticate the user creating the prompt, and distinguish the unique prompt in the case of multiple concurrent prompts from the same user.

Privacy can be an issue: an attacker can try to spam 
```
ln1d6qejxtdg4y5r3zarva:a
ln1d6qejxtdg4y5r3zarva:b
ln1d6qejxtdg4y5r3zarva:c
```
in the hopes of intercepting a prompt.  Since the prompter doesn't know any information (pubkey / address) of the user paying the invoice, (if they did, they wouldn't need to bother with the prompt), they can respond to the wrong person with the invoice info.  The attacker presumably doesn't pay, but does disrupt the invoicing, and learns about what the prompting node is requesting.  The real solution to this is to have an unguessably long prompt identifier like

```ln1d6qejxtdg4y5r3zarva:k6zdkfs4nce4xj0g```

which is 80 bits and they won't be guessing interactively.  That's not good for usability though, which is many cases outweighs the risk of an attacker finding the invoice.  One way to mitigate this is to require the connecting node to provide their own address with some proof of work, and blacklist addresses that have requested an invalid invoice.

The invoice itself can be in JSON.  It could also be fully specified as a binary lndc style message, but it's somewhat in between -- is this control plane or data plane?  JSON might make sense as it straddles the line between human and machine readable.

This can be used for swaps and DLCs as well.  If people are chatting online and want to make a BTC/DOGE swap, after negotiating on IRC one user can post a prompt.  In the case where they know it's going to be copy / pasted and they worry about privacy, they can set it to have a longer invoice identifier.

