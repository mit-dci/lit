package qln

import (
	"fmt"

	"github.com/mit-dci/lit/lnutil"
)

func (nd *LitNode) AddContract() (*lnutil.DlcContract, error) {

	c, err := nd.DlcManager.AddContract()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (nd *LitNode) OfferDlc(peerIdx uint32, cIdx uint64) error {
	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	msg := lnutil.NewDlcOfferMsg(peerIdx, c)
	c.Status = lnutil.ContractStatusOfferedByMe
	c.PeerIdx = peerIdx

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

	msg := lnutil.NewDlcOfferDeclineMsg(c.PeerIdx, 0x01, c.PubKey)
	c.Status = lnutil.ContractStatusDeclined

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.OmniOut <- msg

	return nil
}

func (nd *LitNode) AcceptDlc(cIdx uint64) error {
	c, err := nd.DlcManager.LoadContract(cIdx)
	if err != nil {
		return err
	}

	msg := lnutil.NewDlcOfferAcceptMsg(c.PeerIdx, c.PubKey)
	c.Status = lnutil.ContractStatusAccepted

	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		return err
	}

	nd.OmniOut <- msg

	return nil
}

func (nd *LitNode) DlcOfferHandler(msg lnutil.DlcOfferMsg, peer *RemotePeer) {
	c := new(lnutil.DlcContract)

	c.PeerIdx = peer.Idx
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
	c.OracleR = msg.Contract.OracleR
	c.OracleTimestamp = msg.Contract.OracleTimestamp
	c.PubKey = msg.Contract.PubKey

	err := nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcOfferHandler SaveContract err %s\n", err.Error())
		return
	}
}

func (nd *LitNode) DlcDeclineHandler(msg lnutil.DlcOfferDeclineMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.FindContractByKey(msg.ContractPubKey)
	if err != nil {
		fmt.Printf("DlcDeclineHandler FindContract err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusDeclined
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcDeclineHandler SaveContract err %s\n", err.Error())
		return
	}
}

func (nd *LitNode) DlcAcceptHandler(msg lnutil.DlcOfferAcceptMsg, peer *RemotePeer) {
	c, err := nd.DlcManager.FindContractByKey(msg.ContractPubKey)
	if err != nil {
		fmt.Printf("DlcAcceptHandler FindContract err %s\n", err.Error())
		return
	}

	c.Status = lnutil.ContractStatusAccepted
	err = nd.DlcManager.SaveContract(c)
	if err != nil {
		fmt.Printf("DlcAcceptHandler SaveContract err %s\n", err.Error())
		return
	}

}
