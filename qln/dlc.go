package qln

import (
	"bytes"
	"fmt"

	"github.com/mit-dci/lit/lnutil"
)

func (nd *LitNode) OfferDlc(peerIdx uint32, cIdx uint64) error {
	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	msg := lnutil.NewDlcOfferMsg(peerIdx, c)
	for _, peer := range nd.RemoteCons {
		if peer.Idx == peerIdx {
			copy(c.RemoteNodePub[:], peer.Con.RemotePub.SerializeCompressed())
		}
	}
	c.Status = lnutil.ContractStatusOfferedByMe

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.OmniOut <- msg

	return nil
}

func (nd *LitNode) DeclineDlc(cIdx uint64) error {
	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	peerIdx := uint32(0)
	for _, peer := range nd.RemoteCons {
		if bytes.Equal(c.RemoteNodePub[:], peer.Con.RemotePub.SerializeCompressed()) {
			peerIdx = peer.Idx
			break
		}
	}
	msg := lnutil.NewDlcOfferDeclMsg(peerIdx, 0x01)
	c.Status = lnutil.ContractStatusDeclined

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.OmniOut <- msg

	return nil
}

func (nd *LitNode) DlcOfferHandler(msg lnutil.DlcOfferMsg, peer *RemotePeer) {
	fmt.Println("DlcOfferHandler")

	c := new(lnutil.DlcContract)

	copy(c.RemoteNodePub[:], peer.Con.RemotePub.SerializeCompressed())
	c.Status = lnutil.ContractStatusOfferedToMe

	// Reverse copy from the contract we received
	c.OurFundingAmount = msg.Contract.TheirFundingAmount
	c.TheirFundingAmount = msg.Contract.OurFundingAmount
	c.OurFundingInputs = msg.Contract.TheirFundingInputs
	c.TheirFundingInputs = msg.Contract.OurFundingInputs
	c.OurPayoutPKH = msg.Contract.TheirPayoutPKH
	c.TheirPayoutPKH = msg.Contract.OurPayoutPKH
	c.ValueAllOurs = msg.Contract.ValueAllTheirs
	c.ValueAllTheirs = msg.Contract.ValueAllOurs

	// Copy
	c.CoinType = msg.Contract.CoinType
	c.OracleA = msg.Contract.OracleA
	c.OracleB = msg.Contract.OracleB
	c.OracleQ = msg.Contract.OracleQ
	c.OracleDataFeed = msg.Contract.OracleDataFeed
	c.OracleTimestamp = msg.Contract.OracleTimestamp

	err := nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcOfferHandler SaveContract err %s\n", err.Error())
		return
	}
}

func (nd *LitNode) DlcDeclineHandler(msg lnutil.DlcOfferDeclMsg, peer *RemotePeer) {
	fmt.Println("DlcDeclineHandler")

	contracts, err := nd.DlcManager.ListContracts()
	if err != nil {
		fmt.Printf("DlcDeclineHandler ListContracts err %s\n", err.Error())
		return
	}

	for _, c := range contracts {
		if bytes.Equal(c.RemoteNodePub[:], peer.Con.RemotePub.SerializeCompressed()) {
			c.Status = lnutil.ContractStatusDeclined
			err = nd.DlcManager.SaveContract(c)
			if err != nil {
				fmt.Printf("DlcDeclineHandler SaveContract err %s\n", err.Error())
				return
			}
		}
	}

}
