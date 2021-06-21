package chain

import (
	"errors"

	"github.com/dgraph-io/badger/v3"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
	"google.golang.org/protobuf/proto"
)

/****************************
*
*	chain state machine
*		all pubsub message should be handled here
*
****************************/

var chain_log = logging.Logger("chain")

func HandleTrx(trx *quorumpb.Trx) error {

	verify, err := VerifyTrx(trx)
	if err != nil {
		chain_log.Infof(err.Error())
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
		handleChallenge(trx)
	case quorumpb.TrxType_CHALLENGE_RESP:
		handleReqBlockResp(trx)
	case quorumpb.TrxType_NEW_BLOCK_RESP:
		handleNewBLockResp(trx)
	default:
		err := errors.New("unsupported msg type")
		return err
	}
	return nil
}

func HandleBlock(block *quorumpb.Block) error {
	chain_log.Infof("HandleBlock called")

	if group, ok := GetChainCtx().Groups[block.GroupId]; ok {
		chain_log.Infof("give new block to group")
		err := group.AddBlock(block)
		if err != nil {
			chain_log.Infof(err.Error())
		}
	} else {
		chain_log.Infof("not my block, I don't have the related group")
		if Lucky() {
			chain_log.Infof("save new block to local db")
			GetDbMgr().AddBlock(block)
		}
	}

	return nil
}

func handleTrx(trx *quorumpb.Trx) error {
	chain_log.Infof("handleTrx called")

	if group, ok := GetChainCtx().Groups[trx.GroupId]; ok {
		chain_log.Infof("give new trx to group")
		group.AddTrx(trx)
	}

	return nil
}

func handleReqBlock(trx *quorumpb.Trx) error {
	chain_log.Infof("Handle req block")
	var reqBlockItem quorumpb.ReqBlock
	if err := proto.Unmarshal(trx.Data, &reqBlockItem); err != nil {
		return err
	}

	//check if requester is in group block list
	isBlocked, _ := GetDbMgr().IsBlocked(trx.GroupId, trx.Sender)

	if isBlocked {
		chain_log.Warning("user is blocked by group owner")
		err := errors.New("user auth failed")
		return err
	}

	//check if requested block is in my group and on top
	if group, ok := GetChainCtx().Groups[trx.GroupId]; ok {
		if group.Item.LatestBlockId == reqBlockItem.BlockId {
			chain_log.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_ON_TOP)")
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
							chain_log.Infof("send REQ_NEXT_BLOCK_RESP (BLOCK_IN_TRX)")
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
	chain_log.Infof("handleNextBlockResp called")

	var reqBlockResp quorumpb.ReqBlockResp
	if err := proto.Unmarshal(trx.Data, &reqBlockResp); err != nil {
		return err
	}

	if group, ok := GetChainCtx().Groups[trx.GroupId]; ok {

		if reqBlockResp.Requester != GetChainCtx().PeerId.Pretty() {
			chain_log.Infof("Not asked by me, ignore")
		} else if group.Status == GROUP_CLEAN {
			chain_log.Infof("Group is clean, ignore")
		} else if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_ON_TOP {
			chain_log.Infof("On Group Top, Set Group Status to GROUP_READY")
			group.StopSync()
		} else if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_IN_TRX {
			chain_log.Infof("new block incoming")
			var newBlock quorumpb.Block
			if err := proto.Unmarshal(reqBlockResp.Block, &newBlock); err != nil {
				return err
			}

			topBlock, _ := group.GetTopBlock()
			if valid, _ := IsBlockValid(&newBlock, topBlock); valid {
				chain_log.Infof("block is valid, add it")
				//add block to db
				GetDbMgr().AddBlock(&newBlock)

				//update group block seq map
				group.AddBlock(&newBlock)
			} else {
				chain_log.Infof("block not vaild, skip it")
			}
		}
	} else {
		chain_log.Infof("Can not find group")
	}

	return nil
}

func handleChallenge(trx *quorumpb.Trx) error {
	chain_log.Infof("HandleChallenge called")

	if group, ok := GetChainCtx().Groups[trx.GroupId]; ok {
		chain_log.Infof("give challenge item to group")
		group.UpdateChallenge(trx)
	}

	return nil
}

func handleNewBLockResp(trx *quorumpb.Trx) error {
	chain_log.Infof("HandleNewBlockResp called")

	if group, ok := GetChainCtx().Groups[trx.GroupId]; ok {
		chain_log.Infof("give block response to group")
		group.UpdateNewBlockResp(trx)
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
