import testlib

import time, datetime
import json

import pprint

import requests # pip3 install requests

import codecs

deb_mod = False


def run_t(env, params):
    global deb_mod
    try:

        lit_funding_amt = params[0]
        contract_funding_amt = params[1]
        oracles_number = params[2]
        oracle_value = params[3]
        valueFullyOurs=params[4]
        valueFullyTheirs=params[5]

        feeperbyte = params[6]

        node_to_refund = params[7]

        bc = env.bitcoind

        #------------
        # Create oracles
        #------------

        oracles = []

        for i in range(oracles_number):
            env.new_oracle(1, oracle_value) # publishing interval is 1 second.
            oracles.append(env.oracles[i])

        time.sleep(2)

        #------------
        # Create lits
        #------------

        lit1 = env.lits[0]
        lit2 = env.lits[1]


        pp = pprint.PrettyPrinter(indent=4)


        #------------------------------------------
        if deb_mod:
            print("ADDRESSES BEFORE SEND TO ADDRESS")
            print("LIT1 Addresses")
            print(pp.pprint(lit1.rpc.GetAddresses()))

            print("LIT2 Addresses")
            print(pp.pprint(lit2.rpc.GetAddresses()))

            print("bitcoind Addresses")
            print(pp.pprint(bc.rpc.listaddressgroupings()))
        #------------------------------------------ 


        lit1.connect_to_peer(lit2)
        print("---------------")
        print('Connecting lit1:', lit1.lnid, 'to lit2:', lit2.lnid)

        addr1 = lit1.make_new_addr()
        txid1 = bc.rpc.sendtoaddress(addr1, lit_funding_amt)

        if deb_mod:
            print("Funding TxId lit1: " + str(txid1))

        time.sleep(2)

        addr2 = lit2.make_new_addr()
        txid2 = bc.rpc.sendtoaddress(addr2, lit_funding_amt)

        if deb_mod:
            print("Funding TxId lit2: " + str(txid2))

        time.sleep(2)

        env.generate_block()
        time.sleep(2)


        #------------------------------------------
        if deb_mod:
            print("ADDRESSES AFTER SEND TO ADDRESS")
            print("LIT1 Addresses")
            print(pp.pprint(lit1.rpc.GetAddresses()))

            print("LIT2 Addresses")
            print(pp.pprint(lit2.rpc.GetAddresses()))

            print("bitcoind Addresses")
            print(pp.pprint(bc.rpc.listaddressgroupings()))
        #------------------------------------------    


        print("Funding")
        bals1 = lit1.get_balance_info()  
        print('new lit1 balance:', bals1['TxoTotal'], 'in txos,', bals1['ChanTotal'], 'in chans')
        bal1sum = bals1['TxoTotal'] + bals1['ChanTotal']
        print('  = sum ', bal1sum)

        print(lit_funding_amt)

        lit_funding_amt *= 100000000        # to satoshi


        bals2 = lit2.get_balance_info()
        print('new lit2 balance:', bals2['TxoTotal'], 'in txos,', bals2['ChanTotal'], 'in chans')
        bal2sum = bals2['TxoTotal'] + bals2['ChanTotal']
        print('  = sum ', bal2sum) 


        assert bal1sum == lit_funding_amt, "Funding lit1 does not works"
        assert bal2sum == lit_funding_amt, "Funding lit2 does not works"
      
        # #------------
        # # Add oracles
        # #------------

        res = lit1.rpc.ListOracles()
        assert len(res) != 0, "Initial lis of oracles must be empty"
        
        oracles_pubkey = []
        oidxs = []
        datasources = []

        for oracle in oracles:
            opk = json.loads(oracle.get_pubkey())
            oracles_pubkey.append(opk)

            oidx = lit1.rpc.AddOracle(Key=opk["A"], Name=opk["A"])["Oracle"]["Idx"]
            oidxs.append(oidx)
            lit2.rpc.AddOracle(Key=opk["A"], Name=opk["A"])["Oracle"]["Idx"]

            datasources.append(json.loads(oracle.get_datasources()))


        # #------------
        # # Now we have to create a contract in the lit1 node.
        # #------------

        contract = lit1.rpc.NewContract()

        res = lit1.rpc.ListContracts()
        assert len(res["Contracts"]) == 1, "ListContracts does not works"


        res = lit1.rpc.GetContract(Idx=1)
        assert res["Contract"]["Idx"] == 1, "GetContract does not works"
                

        res = lit1.rpc.SetContractOraclesNumber(CIdx=contract["Contract"]["Idx"], OraclesNumber=oracles_number)
        assert res["Success"], "SetContractOraclesNumber does not works"

        res = lit1.rpc.SetContractOracle(CIdx=contract["Contract"]["Idx"], OIdx=oidxs)
        assert res["Success"], "SetContractOracle does not works"


        # Since the oracle publishes data every 1 second (we set this time above), 
        # we increase the time for a point by 3 seconds.

        settlement_time = int(time.time()) + 3

        # dlc contract settime
        res = lit1.rpc.SetContractSettlementTime(CIdx=contract["Contract"]["Idx"], Time=settlement_time)
        assert res["Success"], "SetContractSettlementTime does not works"

        # we set settlement_time equal to refundtime, actually the refund transaction will be valid.
        res = lit1.rpc.SetContractRefundTime(CIdx=contract["Contract"]["Idx"], Time=settlement_time)
        assert res["Success"], "SetContractRefundTime does not works"

        res = lit1.rpc.ListContracts()
        assert res["Contracts"][contract["Contract"]["Idx"] - 1]["OracleTimestamp"] == settlement_time, "SetContractSettlementTime does not match settlement_time"

        decode_hex = codecs.getdecoder("hex_codec")
        brpoints = []
        rpoints = []
        for oracle, datasource in zip(oracles, datasources):
            res = oracle.get_rpoint(datasource[0]["id"], settlement_time)
            print(res)
            b_RPoint = decode_hex(json.loads(res)['R'])[0]
            RPoint = [elem for elem in b_RPoint]
            brpoints.append(RPoint)
            rpoints.append(res)

        res = lit1.rpc.SetContractRPoint(CIdx=contract["Contract"]["Idx"], RPoint=brpoints)
        assert res["Success"], "SetContractRpoint does not works"

        lit1.rpc.SetContractCoinType(CIdx=contract["Contract"]["Idx"], CoinType = 257)
        res = lit1.rpc.GetContract(Idx=contract["Contract"]["Idx"])
        assert res["Contract"]["CoinType"] == 257, "SetContractCoinType does not works"


        lit1.rpc.SetContractFeePerByte(CIdx=contract["Contract"]["Idx"], FeePerByte = feeperbyte)
        res = lit1.rpc.GetContract(Idx=contract["Contract"]["Idx"])
        assert res["Contract"]["FeePerByte"] == feeperbyte, "SetContractFeePerByte does not works"        

        ourFundingAmount = contract_funding_amt
        theirFundingAmount = contract_funding_amt

        lit1.rpc.SetContractFunding(CIdx=contract["Contract"]["Idx"], OurAmount=ourFundingAmount, TheirAmount=theirFundingAmount)
        res = lit1.rpc.GetContract(Idx=contract["Contract"]["Idx"])
        assert res["Contract"]["OurFundingAmount"] == ourFundingAmount, "SetContractFunding does not works"
        assert res["Contract"]["TheirFundingAmount"] == theirFundingAmount, "SetContractFunding does not works"

        res = lit1.rpc.SetContractDivision(CIdx=contract["Contract"]["Idx"], ValueFullyOurs=valueFullyOurs, ValueFullyTheirs=valueFullyTheirs)
        assert res["Success"], "SetContractDivision does not works"
        
        time.sleep(3)
  
        res = lit1.rpc.ListConnections()
        print(res)

        res = lit1.rpc.OfferContract(CIdx=contract["Contract"]["Idx"], PeerIdx=lit1.get_peer_id(lit2))
        assert res["Success"], "OfferContract does not works"

        time.sleep(3)
       
        res = lit2.rpc.ContractRespond(AcceptOrDecline=True, CIdx=1)
        assert res["Success"], "ContractRespond on lit2 does not works"

        time.sleep(3)

        #------------------------------------------
        
        if deb_mod:
            print("ADDRESSES AFTER CONTRACT RESPOND")
            print("LIT1 Addresses")
            print(pp.pprint(lit1.rpc.GetAddresses()))

            print("LIT2 Addresses")
            print(pp.pprint(lit2.rpc.GetAddresses()))

            print("bitcoind Addresses")
            print(pp.pprint(bc.rpc.listaddressgroupings()))


        # #------------------------------------------  

        env.generate_block()
        time.sleep(2)


        print("Accept")
        bals1 = lit1.get_balance_info()  
        print('new lit1 balance:', bals1['TxoTotal'], 'in txos,', bals1['ChanTotal'], 'in chans')
        bal1sum = bals1['TxoTotal'] + bals1['ChanTotal']
        print('  = sum ', bal1sum)

        lit1_bal_after_accept = (lit_funding_amt - ourFundingAmount) - (126*feeperbyte)
        

        bals2 = lit2.get_balance_info()
        print('new lit2 balance:', bals2['TxoTotal'], 'in txos,', bals2['ChanTotal'], 'in chans')
        bal2sum = bals2['TxoTotal'] + bals2['ChanTotal']
        print('  = sum ', bal2sum)   

        lit2_bal_after_accept = (lit_funding_amt - theirFundingAmount) - (126*feeperbyte)


        assert bal1sum == lit1_bal_after_accept, "lit1 Balance after contract accept does not match"
        assert bal2sum == lit2_bal_after_accept, "lit2 Balance after contract accept does not match"        

        time.sleep(2)

        res = env.lits[node_to_refund].rpc.RefundContract(CIdx=1)

        time.sleep(2)
        env.generate_block()
        time.sleep(1)
        env.generate_block()
        time.sleep(1)


        if deb_mod:
            print("ADDRESSES AFTER CONTRACT Refund")
            print("LIT1 Addresses")
            print(pp.pprint(lit1.rpc.GetAddresses()))

            print("LIT2 Addresses")
            print(pp.pprint(lit2.rpc.GetAddresses()))

            print("bitcoind Addresses")
            print(pp.pprint(bc.rpc.listaddressgroupings()))



        print("Refund")
        bals1 = lit1.get_balance_info()  
        print('new lit1 balance:', bals1['TxoTotal'], 'in txos,', bals1['ChanTotal'], 'in chans')
        bal1sum = bals1['TxoTotal'] + bals1['ChanTotal']
        print('  = sum ', bal1sum)


        bals2 = lit2.get_balance_info()
        print('new lit2 balance:', bals2['TxoTotal'], 'in txos,', bals2['ChanTotal'], 'in chans')
        bal2sum = bals2['TxoTotal'] + bals2['ChanTotal']
        print('  = sum ', bal2sum) 

        assert bal1sum == 99976400, "lit1 After Refund balance does not match."
        assert bal2sum == 99976400, "lit2 After Refund balance does not match."        


        #-----------------------------------------------
        # Send 10000 sat from lit1 to lit2

        print("Address to send: ")
        print(lit2.rpc.GetAddresses()['WitAddresses'][0])

        res = lit1.rpc.Send(DestAddrs=[lit2.rpc.GetAddresses()['WitAddresses'][0]], Amts=[99960240])

        time.sleep(1)
        env.generate_block()
        time.sleep(2)

        print("After Spend")
        bals1 = lit1.get_balance_info()  
        print('new lit1 balance:', bals1['TxoTotal'], 'in txos,', bals1['ChanTotal'], 'in chans')
        bal1sum = bals1['TxoTotal'] + bals1['ChanTotal']
        print('  = sum ', bal1sum)


        bals2 = lit2.get_balance_info()
        print('new lit2 balance:', bals2['TxoTotal'], 'in txos,', bals2['ChanTotal'], 'in chans')
        bal2sum = bals2['TxoTotal'] + bals2['ChanTotal']
        print('  = sum ', bal2sum) 

        assert bal1sum == 0, "lit1 After Spend balance does not match."
        assert bal2sum == 199936640, "lit2 After Spend balance does not match."
        
        #------------------------------------------------

        if deb_mod:

            print("==========================================================")
            print('Print Blockchain Info')
            print("==========================================================")


            best_block_hash = bc.rpc.getbestblockhash()
            bb = bc.rpc.getblock(best_block_hash)
            print(bb)
            print("bb['height']: " + str(bb['height']))

            print("Balance from RPC: " + str(bc.rpc.getbalance()))

            # batch support : print timestamps of blocks 0 to 99 in 2 RPC round-trips:
            commands = [ [ "getblockhash", height] for height in range(bb['height'] + 1) ]
            block_hashes = bc.rpc.batch_(commands)
            blocks = bc.rpc.batch_([ [ "getblock", h ] for h in block_hashes ])
            block_times = [ block["time"] for block in blocks ]
            print(block_times)

            print('--------------------')

            for b in blocks:
                print("--------BLOCK--------")
                print(b)
                tx = b["tx"]
                try:

                    for i in range(len(tx)):
                        print("--------TRANSACTION--------")
                        rtx = bc.rpc.getrawtransaction(tx[i])
                        print(rtx)
                        decoded = bc.rpc.decoderawtransaction(rtx)
                        pp.pprint(decoded)
                except BaseException as be:
                    print(be)
                print('--------')    

    except BaseException as be:
        raise be


# ====================================================================================
# ====================================================================================  



def forward(env):
    
    oracles_number = 3
    oracle_value = 10
    node_to_settle = 0

    valueFullyOurs=10
    valueFullyTheirs=20

    lit_funding_amt =      1     # 1 BTC
    contract_funding_amt = 10000000     # satoshi

    feeperbyte = 80

    params = [lit_funding_amt, contract_funding_amt, oracles_number, oracle_value, valueFullyOurs, valueFullyTheirs, feeperbyte, 0]

    run_t(env, params)



def reverse(env):

    oracles_number = 3
    oracle_value = 10
    node_to_settle = 0

    valueFullyOurs=10
    valueFullyTheirs=20

    lit_funding_amt =      1     # 1 BTC
    contract_funding_amt = 10000000     # satoshi

    feeperbyte = 80

    params = [lit_funding_amt, contract_funding_amt, oracles_number, oracle_value, valueFullyOurs, valueFullyTheirs, feeperbyte, 1]

    run_t(env, params)