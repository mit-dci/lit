import testlib

# So the test roughly goes as follows:
#  Start lit up normally, generate 10 blocks using bitcoind's generate command
# Store the first hash and invalidate that, creating a 9 block re-org. Then
# Generate block by block and see if lit reorgs. Lit should reorg only after
# we produce 9 blocks
def run_test(env):
    print("REUNNINSA")
    bc = env.bitcoind
    lit1 = env.lits[0]
    bc.rpc.generate(11)
    tip = bc.rpc.getblockcount()
    print("TIP", tip)
    targetblockhash = bc.rpc.getblockhash(tip - 9)
    # TODO: Stop and start lit again here
    lit1.shutdown()
    lit1.start()
    bc.rpc.invalidateblock(targetblockhash)
    # Tip now at block 501
    # Now check where lit's latest block is, it should be at 511 still
    assert lit1.get_sync_height() == 511, "Reorg error, block height should be 511"
    bc.rpc.generate(1)
    print("CHK", lit1.get_sync_height())
    assert lit1.get_sync_height() == 511, "Reorg error, block height should be 511"
    print("CHK2", lit1.get_sync_height())
    bc.rpc.generate(11)
    print("NEWTIP", bc.rpc.getblockcount())
    assert lit1.get_sync_height() == 512, "Reorg error, block height should be 511"
    assert 1 > 0
