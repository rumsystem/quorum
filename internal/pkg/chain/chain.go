package chain

import (
	"encoding/json"
	"errors"
	//"fmt"
	badger "github.com/dgraph-io/badger/v3"
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

	verify, err := VerifyTrx(trxMsg)
	if err != nil {
		glog.Infof(err.Error())
		return err
	}

	if !verify {
		err := errors.New("Can not verify trx")
		return err
	}

	switch trxMsg.MsgType {
	case REQ_SIGN:
		handleReqSign(trxMsg)
	case REQ_SIGN_RESP:
		handleReqSignResp(trxMsg)
	case ADD_NEW_BLOCK:
		handleNewBlock(trxMsg)
	case ADD_NEW_BLOCK_RESP:
		handleNewBlockResp(trxMsg)
	case REQ_NEXT_BLOCK:
		handleNextBlock(trxMsg)
	case REQ_NEXT_BLOCK_RESP:
		handleNextBlockResp(trxMsg)
	case PEER_ANNOUNCE:
		handlePeerAnnounce(trxMsg)
	default:
		err := errors.New("unsupported msg type")
		return err
	}

	return nil
}

func handleReqSign(trxMsg TrxMsg) error {
	glog.Infof("handleReqSign called")

	var reqSign ReqSign
	if err := json.Unmarshal(trxMsg.Data, &reqSign); err != nil {
		return err
	}

	if lucky := Lucky(); lucky {
		glog.Infof("sign it and send ReqSignResp msg")
		var trxMsg2 TrxMsg
		trxMsg2, _ = CreateTrxMsgReqSignResp(trxMsg, reqSign)
		if jsonBytes, err := json.Marshal(trxMsg2); err != nil {
			return err
		} else {
			GetChainCtx().PublicTopic.Publish(GetChainCtx().Ctx, jsonBytes)
		}
	}

	return nil
}

func handleReqSignResp(trxMsg TrxMsg) error {
	glog.Infof("handleReqSignResp called")

	var reqSignResp ReqSignResp
	if err := json.Unmarshal(trxMsg.Data, &reqSignResp); err != nil {
		return err
	}

	if reqSignResp.Requester != GetChainCtx().PeerId.Pretty() {
		//glog.Infof("Not requested by me, ignore")
		return nil
	}

	trx, err := GetDbMgr().GetTrx(reqSignResp.ReqTrxId)
	if err != nil {
		return err
	}

	hash := string(reqSignResp.Hash)
	wsign := string(reqSignResp.WitnessSign)
	consensusString := "witness?=" + reqSignResp.Witness + "/hash?=" + hash + "/wsign?=" + wsign

	trx.Consensus = append(trx.Consensus, consensusString)

	if err := GetDbMgr().UpdTrxCons(trx, consensusString); err != nil {
		return err
	}

	if len(trx.Consensus) < GetChainCtx().TrxSignReq { //check if we have enough signature
		glog.Infof("Wait more signature to come")
		return nil
	} else if groupItem, OK := GetChainCtx().Groups[trxMsg.GroupId]; OK {
		//Get topblock and create a new block to include trx
		topBlock, _ := groupItem.GetTopBlock()
		newBlock := CreateBlock(topBlock, trx)

		//Create NEW_BLOCK msg and send it out
		newBlockTrxMsg, _ := CreateTrxNewBlock(newBlock)
		jsonBytes, _ := json.Marshal(newBlockTrxMsg)
		GetChainCtx().PublicTopic.Publish(GetChainCtx().Ctx, jsonBytes)

		//Give new block to group
		groupItem.AddBlock(newBlock)
	} else {
		glog.Infof("Can not find group")
	}

	return nil
}

func handleNewBlock(trxMsg TrxMsg) error {
	glog.Infof("handleNewBlock called")

	var newBlock NewBlock
	if err := json.Unmarshal(trxMsg.Data, &newBlock); err != nil {
		return err
	}

	var block Block
	if err := json.Unmarshal(newBlock.Data, &block); err != nil {
		return err
	}

	sendResp := true
	if group, ok := GetChainCtx().Groups[block.GroupId]; ok {
		glog.Infof("give new block to group")
		group.AddBlock(block)
	} else {
		glog.Infof("not my block, I don't have the related group")
		if Lucky() {
			glog.Infof("save new block to local db")
			GetDbMgr().AddBlock(block)
		} else {
			sendResp = false
		}
	}

	//send NewBlockResp msg
	if sendResp {
		glog.Infof("send Add_NEW_BLOCK_RESP")
		newBlockRespMsg, _ := CreateTrxNewBlockResp(block)
		jsonBytes, _ := json.Marshal(newBlockRespMsg)
		GetChainCtx().PublicTopic.Publish(GetChainCtx().Ctx, jsonBytes)
	}

	return nil
}

func handleNewBlockResp(trxMsg TrxMsg) error {
	glog.Infof("handleNewBlockResp called")

	//know block is saved
	//remove local req
	//update block status
	return nil
}

func handleNextBlock(trxMsg TrxMsg) error {
	glog.Infof("handleNextBlock called...")

	var reqNextBlock ReqNextBlock
	if err := json.Unmarshal(trxMsg.Data, &reqNextBlock); err != nil {
		return err
	}

	//check if requested block is in my group and on top
	if group, ok := GetChainCtx().Groups[trxMsg.GroupId]; ok {
		if group.Item.LatestBlockId == reqNextBlock.BlockId {
			glog.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_ON_TOP)")
			var emptyBlock Block
			emptyBlock.GroupId = trxMsg.GroupId
			nextBlockRespMsg, _ := CreateTrxReqNextBlockResp(BLOCK_ON_TOP, trxMsg.Sender, emptyBlock)
			jsonBytes, _ := json.Marshal(nextBlockRespMsg)
			GetChainCtx().PublicTopic.Publish(GetChainCtx().Ctx, jsonBytes)
			return nil
		}

		//otherwise, check blockDB, if I have the block requested, send it out by publish
		err := GetDbMgr().BlockDb.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchSize = 10
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				//k := item.Key()
				err := item.Value(func(v []byte) error {
					var block Block
					if err := json.Unmarshal(v, &block); err != nil {
						return err
					}

					if block.PrevBlockId == reqNextBlock.BlockId {
						glog.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)")
						nextBlockRespMsg, _ := CreateTrxReqNextBlockResp(BLOCK_IN_TRX, trxMsg.Sender, block)
						jsonBytes, _ := json.Marshal(nextBlockRespMsg)
						GetChainCtx().PublicTopic.Publish(GetChainCtx().Ctx, jsonBytes)
					}
					return nil
				})

				if err != nil {
					return err
				}
			}
			return nil
		})

		if err != nil {
			glog.Fatalf(err.Error())
		}
	}

	return nil
}

func handleNextBlockResp(trxMsg TrxMsg) error {
	glog.Infof("handleNextBlockResp called")

	var reqNextBlockResp ReqNextBlockResp
	if err := json.Unmarshal(trxMsg.Data, &reqNextBlockResp); err != nil {
		return err
	}

	if group, ok := GetChainCtx().Groups[trxMsg.GroupId]; ok {

		if reqNextBlockResp.Requester != GetChainCtx().PeerId.Pretty() {
			glog.Infof("Not asked by me, ignore")
		} else if group.Status == GROUP_CLEAN {
			glog.Infof("Group is clean, ignore")
		} else if reqNextBlockResp.Response == BLOCK_ON_TOP {
			glog.Infof("On Group Top, Set Group Status to GROUP_READY")
			group.StopSync()
		} else if reqNextBlockResp.Response == BLOCK_IN_TRX {
			glog.Infof("new block incoming")
			var newBlock Block
			if err := json.Unmarshal(reqNextBlockResp.Block, &newBlock); err != nil {
				return err
			}

			topBlock, _ := group.GetTopBlock()
			if valid, _ := IsBlockValid(newBlock, topBlock); valid {
				glog.Infof("block is valid, add it")
				//add block to db
				GetDbMgr().AddBlock(newBlock)

				//update group block seq map
				group.AddBlock(newBlock)
			}
		}
	} else {
		glog.Infof("Can not find group")
	}

	return nil
}

func handlePeerAnnounce(trxMsg TrxMsg) error {
	glog.Infof("handlePeerAnnounce called")
	return nil
}
