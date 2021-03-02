package chain

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
)

const CONSENSUS uint8 = 1

/****************************
*
*	chain state machine
*		all pubsub message should be handled here
*
****************************/

func handleTrxMsg(trxMsg TrxMsg) error {
	//glog.Infof(trxMsg.MsgType)
	glog.Infof(trxMsg.TrxId)
	switch trxMsg.MsgType {
	case REQ_SIGN:
		handleReqSign(trxMsg)
	case REQ_SIGN_RESP:
		handleReqSignResp(trxMsg)
	case NEW_BLOCK:
		handleNewBlock(trxMsg)
	case NEW_BLOCK_RESP:
		handleNewBlockResp(trxMsg)
	case REQ_NEXT_BLOCK:
		handleNextBlock(trxMsg)
	case REQ_NEXT_BLOCK_RESP:
		handleNextBlockResp(trxMsg)
	case PEER_ANNOUNCE:
		handlePeerAnnounce(trxMsg)
	default:
		err := errors.New("unsupported msg typ")
		return err
	}

	return nil
}

func handleReqSign(trxMsg TrxMsg) error {
	glog.Infof("handleReqSign called")

	var regSign TrxRegSign
	err := json.Unmarshal(trxMsg.Data, &regSign)

	if err != nil {
		return err
	}

	glog.Infof("trxId::" + trxMsg.TrxId)
	glog.Infof("Sender::" + regSign.Sender)

	if regSign.Sender == GetContext().PeerId.Pretty() {
		glog.Infof("msg from myself, ignore")
	} else {
		lucky := Lucky()
		if lucky {
			glog.Infof("sign it and send RegSignResp msg")
			//Verify trxMsg signature, if correct, sign it and publish
			//If failed, do nothing
			var trxMsg2 TrxMsg
			trxMsg2, _ = CreateTrxMsgRegSignResp(trxMsg.TrxId, GetContext().PeerId.Pretty(), regSign)
			jsonBytes, _ := json.Marshal(trxMsg2)
			GetContext().PublicTopic.Publish(GetContext().Ctx, jsonBytes)
		}
	}

	return nil
}

//test only, should be put into global config
const TRX_SIGNATURE_REQ_COUNT int = 1

func handleReqSignResp(trxMsg TrxMsg) error {
	glog.Infof("handleReqSign called")

	var respSign TrxReqSignResp
	err := json.Unmarshal(trxMsg.Data, &respSign)

	if err != nil {
		return err
	}

	glog.Info("trxId::" + trxMsg.TrxId)
	glog.Info("witness::" + respSign.Witness)

	if respSign.Witness == GetContext().PeerId.Pretty() {
		glog.Infof("msg from myself, ignore")
	} else {
		glog.Infof("update trx consense")
		witnessSignature := string(respSign.Signature)
		consensusString := respSign.Witness + "::signautre=" + witnessSignature
		UpdTrxCons(respSign.ReqTrxId, consensusString)

		trx, _ := GetTrx(respSign.ReqTrxId)
		if len(trx.Consensus) >= TRX_SIGNATURE_REQ_COUNT {
			glog.Info("trx signed!!! create block now")
			topBlock, _ := GetTopBlock(TestGroupId)
			newBlock := CreateBlock(topBlock, trx)
			fmt.Printf("%v", newBlock)
			//sent it out
		} else {
			glog.Info("wait for more signatures")
		}
	}

	return nil
}

func handleNewBlock(trxMsg TrxMsg) error {
	glog.Infof("handleNewBlock called")

	if Lucky() {
		glog.Infof("save block")
		glog.Infof("send NewBlockResp")
	}
	return nil
}

func handleNewBlockResp(trxMsg TrxMsg) error {
	glog.Infof("handleNewBlockResp called")

	//know block is saved
	//do nothing??? or handle timeout?

	return nil
}

func handleNextBlock(trxMsg TrxMsg) error {
	glog.Infof("handleNextBlock called")

	//check blockDB, if I have the block requested, send it out by publish
	return nil
}

func handleNextBlockResp(trxMsg TrxMsg) error {
	glog.Infof("handleNextBlockResp called")

	//verify block, verify all trx signatures
	//if valid, add it to my local block db and block list
	//refresh content list

	return nil
}

func handlePeerAnnounce(trxMsg TrxMsg) error {
	glog.Infof("handlePeerAnnounce called")
	return nil
}
