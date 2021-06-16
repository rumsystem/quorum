package chain

import (
	"errors"

	"github.com/dgraph-io/badger/v3"
	"github.com/golang/glog"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

const CONSENSUS uint8 = 1

/****************************
*
*	chain state machine
*		all pubsub message should be handled here
*
****************************/

func HandleTrx(trx *quorumpb.Trx) error {

	verify, err := VerifyTrx(trx)
	if err != nil {
		glog.Infof(err.Error())
		return err
	}

	if !verify {
		err := errors.New("Can not verify trx")
		return err
	}

	switch trx.Type {
	case quorumpb.TrxType_AUTH:
		handleTrx(trx)
	case quorumpb.TrxType_POST:
		handleTrx(trx)
	case quorumpb.TrxType_REQ_BLOCK:
		handleReqBlock(trx)
	case quorumpb.TrxType_REQ_BLOCK_RESP:
		handleReqBlockResp(trx)
	case quorumpb.TrxType_CHALLENGE:
		HandleChallenge(trx)
	case quorumpb.TrxType_CHALLENGE_RESP:
		handleReqBlockResp(trx)
	default:
		err := errors.New("unsupported msg type")
		return err
	}
	return nil
}

func HandleBlock(block *quorumpb.Block) error {
	glog.Infof("HandleBlock called")

	if group, ok := GetChainCtx().Groups[block.GroupId]; ok {
		glog.Infof("give new block to group")
		err := group.AddBlock(block)
		if err != nil {
			glog.Infof(err.Error())
		}
	} else {
		glog.Infof("not my block, I don't have the related group")
		if Lucky() {
			glog.Infof("save new block to local db")
			GetDbMgr().AddBlock(block)
		}
	}

	return nil
}

func handleTrx(trx *quorumpb.Trx) error {
	glog.Infof("handleTrx called")

	if group, ok := GetChainCtx().Groups[trx.GroupId]; ok {
		glog.Infof("give new trx to group")
		group.AddTrx(trx)
	}

	return nil
}

func handleReqBlock(trx *quorumpb.Trx) error {
	glog.Infof("Handle req block")
	var reqBlockItem quorumpb.ReqBlock
	if err := proto.Unmarshal(trx.Data, &reqBlockItem); err != nil {
		return err
	}

	//check if requester is in group block list
	isBlocked, _ := GetDbMgr().IsBlocked(trx.GroupId, trx.Sender)

	if isBlocked {
		glog.Warning("user is blocked by group owner")
		err := errors.New("user auth failed")
		return err
	}

	//check if requested block is in my group and on top
	if group, ok := GetChainCtx().Groups[trx.GroupId]; ok {
		if group.Item.LatestBlockId == reqBlockItem.BlockId {
			glog.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_ON_TOP)")
			var emptyblock quorumpb.Block
			err := SendReqBlockResp(trx, &reqBlockItem, &emptyblock)

			if err != nil {
				return err
			}
		} else {
			err := GetDbMgr().Db.View(func(txn *badger.Txn) error {
				opts := badger.DefaultIteratorOptions
				opts.PrefetchSize = 10
				it := txn.NewIterator(opts)
				defer it.Close()
				for it.Seek([]byte(BLK_PREFIX)); it.ValidForPrefix([]byte(BLK_PREFIX)); it.Next() {
					item := it.Item()
					err := item.Value(func(v []byte) error {
						var block quorumpb.Block
						if err := proto.Unmarshal(v, &block); err != nil {
							return err
						}

						if block.PrevBlockId == reqBlockItem.BlockId {
							glog.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)")
							err := SendReqBlockResp(trx, &reqBlockItem, &block)
							return err

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
				return err
			}
		}
	}

	return nil
}

func handleReqBlockResp(trx *quorumpb.Trx) error {
	glog.Infof("handleNextBlockResp called")

	var reqBlockResp quorumpb.ReqBlockResp
	if err := proto.Unmarshal(trx.Data, &reqBlockResp); err != nil {
		return err
	}

	if group, ok := GetChainCtx().Groups[trx.GroupId]; ok {

		if reqBlockResp.Requester != GetChainCtx().PeerId.Pretty() {
			glog.Infof("Not asked by me, ignore")
		} else if group.Status == GROUP_CLEAN {
			glog.Infof("Group is clean, ignore")
		} else if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_ON_TOP {
			glog.Infof("On Group Top, Set Group Status to GROUP_READY")
			group.StopSync()
		} else if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_IN_TRX {
			glog.Infof("new block incoming")
			var newBlock quorumpb.Block
			if err := proto.Unmarshal(reqBlockResp.Block, &newBlock); err != nil {
				return err
			}

			topBlock, _ := group.GetTopBlock()
			if valid, _ := IsBlockValid(&newBlock, topBlock); valid {
				glog.Infof("block is valid, add it")
				//add block to db
				GetDbMgr().AddBlock(&newBlock)

				//update group block seq map
				group.AddBlock(&newBlock)
			} else {
				glog.Infof("block not vaild, skip it")
			}
		}
	} else {
		glog.Infof("Can not find group")
	}

	return nil
}

func HandleChallenge(trx *quorumpb.Trx) error {
	glog.Infof("HandleChallenge called")

	if group, ok := GetChainCtx().Groups[trx.GroupId]; ok {
		glog.Infof("give challenge item to group")
		group.UpdateChallenge(trx)
	}

	return nil
}

func SendReqBlockResp(trx *quorumpb.Trx, req *quorumpb.ReqBlock, block *quorumpb.Block) error {
	var reqBlockRespItem quorumpb.ReqBlockResp
	reqBlockRespItem.Result = quorumpb.ReqBlkResult_BLOCK_ON_TOP
	reqBlockRespItem.Provider = GetChainCtx().PeerId.Pretty()
	reqBlockRespItem.Requester = trx.Sender
	reqBlockRespItem.GroupId = trx.GroupId
	reqBlockRespItem.BlockId = req.BlockId

	pbBytesBlock, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	reqBlockRespItem.Block = pbBytesBlock

	bItemBytes, err := proto.Marshal(&reqBlockRespItem)
	if err != nil {
		return err
	}

	//send ask next block trx out
	respTrx, err := CreateTrx(quorumpb.TrxType_REQ_BLOCK_RESP, reqBlockRespItem.GroupId, bItemBytes)
	if err != nil {
		return err
	}

	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(respTrx)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_TRX
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	return GetChainCtx().GroupTopicPublish(trx.GroupId, pkgBytes)
}
