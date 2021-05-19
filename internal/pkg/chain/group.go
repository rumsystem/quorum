package chain

import (
	"errors"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
	"time"

	"github.com/golang/glog"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

type GroupStatus int8

const (
	GROUP_CLEAN = 0
	GROUP_DIRTY = 1
)

type Group struct {
	Item          *quorumpb.GroupItem
	AskNextTicker *time.Ticker
	TickerDone    chan bool
	Status        GroupStatus
}

func (grp *Group) init(item *quorumpb.GroupItem) error {
	grp.Item = item
	grp.AskNextTicker = time.NewTicker(1000 * time.Millisecond)
	grp.TickerDone = make(chan bool)
	return nil
}

func (grp *Group) Teardown() {
	if grp.Status == GROUP_DIRTY {
		grp.stopAskNextBlock()
	}
}

//Start sync group
func (grp *Group) StartSync() error {
	glog.Infof("Group %s start syncing", grp.Item.GroupId)
	grp.Status = GROUP_DIRTY
	grp.startAskNextBlock()
	return nil
}

//Stop sync group
func (grp *Group) StopSync() error {
	glog.Infof("Group stop sync")
	grp.Status = GROUP_CLEAN
	grp.stopAskNextBlock()
	return nil
}

func (grp *Group) AddBlock(block *quorumpb.Block) error {
	//verify block
	if block.BlockNum != 0 {
		topBlock, err := grp.GetTopBlock()

		if err != nil {
			return err
		}

		valid, _ := IsBlockValid(block, topBlock)
		if !valid {
			err := errors.New("Invalid block")
			return err
		}
	}

	//save block to local db
	err := GetDbMgr().AddBlock(block)
	if err != nil {
		return err
	}

	err = GetDbMgr().UpdBlkSeq(block)
	if err != nil {
		return err
	}

	err = GetDbMgr().AddGrpCtnt(block)
	if err != nil {
		return err
	}

	grp.Item.LatestBlockNum = block.BlockNum
	grp.Item.LatestBlockId = block.Cid
	grp.Item.LastUpdate = time.Now().UnixNano()

	//update local db
	dbMgr.UpdGroup(grp.Item)

	return nil
}

func (grp *Group) GetTopBlock() (*quorumpb.Block, error) {
	return GetDbMgr().GetBlock(grp.Item.LatestBlockId)
}

func (grp *Group) GetBlockId(blockNum int64) (string, error) {
	return GetDbMgr().GetBlkId(blockNum, grp.Item.GroupId)
}

func (grp *Group) CreateGrp(item *quorumpb.GroupItem) error {
	err := grp.init(item)
	if err != nil {
		return err
	}

	err = GetDbMgr().AddBlock(item.GenesisBlock)
	if err != nil {
		return err
	}
	return GetDbMgr().AddGroup(grp.Item)
}

func (grp *Group) DelGrp() error {
	pubkeybytes, err := p2pcrypto.MarshalPublicKey(GetChainCtx().PublicKey)
	if err != nil {
		return err
	}

	if grp.Item.OwnerPubKey != p2pcrypto.ConfigEncodeKey(pubkeybytes) {
		err := errors.New("You can not 'delete' group created by others, use 'leave' instead")
		return err
	}

	return GetDbMgr().RmGroup(grp.Item)
}

func (grp *Group) LeaveGrp() error {
	pubkeybytes, err := p2pcrypto.MarshalPublicKey(GetChainCtx().PublicKey)
	if err != nil {
		return err
	}

	if grp.Item.OwnerPubKey == p2pcrypto.ConfigEncodeKey(pubkeybytes) {
		err := errors.New("Group creator can not leave the group they created, use 'delete' instead")
		return err
	}

	return GetDbMgr().RmGroup(grp.Item)
}

//Add Content to Group
func (grp *Group) Post(content *quorumpb.Object) (string, error) {
	var trx quorumpb.Trx
	var trxMsg *quorumpb.TrxMsg

	encodedcontent, err := proto.Marshal(content)
	if err != nil {
		return "", err
	}

	trxMsg, _ = CreateTrxMsgReqSign(grp.Item.GroupId, encodedcontent)
	trx.Msg = trxMsg
	trx.Data = encodedcontent
	var cons []string
	trx.Consensus = cons

	dbMgr.AddTrx(&trx)

	pbBytes, err := proto.Marshal(trxMsg)
	if err != nil {
		return "INVALID_TRX", err
	}

	err = GetChainCtx().GroupTopicPublish(trxMsg.GroupId, pbBytes)
	return trxMsg.TrxId, err
}

//Load groupItem from DB
func (grp *Group) GetContent(upperCount, lowerCount uint64) ([]*quorumpb.GroupContentItem, error) {
	var ctnList []*quorumpb.GroupContentItem
	return ctnList, nil
}

//ask next block
func (grp *Group) startAskNextBlock() {
	//send ask_next_block every 1 sec till get "on_top response"
	go func() {
		for {
			select {
			case <-grp.TickerDone:
				return
			case t := <-grp.AskNextTicker.C:
				glog.Infof("Ask NEXT_BLOCK " + t.UTC().String())
				//send ask next block msg out
				topBlock, err := grp.GetTopBlock()
				if err != nil {
					glog.Fatalf(err.Error())
				}

				askNextMsg, err := CreateTrxReqNextBlock(topBlock)
				if err != nil {
					glog.Fatalf(err.Error())
				}

				pbBytes, err := proto.Marshal(askNextMsg)
				if err != nil {
					glog.Fatalf(err.Error())
				}

				err = GetChainCtx().GroupTopicPublish(askNextMsg.GroupId, pbBytes)
				if err != nil {
					glog.Fatalf(err.Error())
				}
			}
		}
	}()
}

func (grp *Group) stopAskNextBlock() {
	grp.AskNextTicker.Stop()
	grp.TickerDone <- true
	grp.Status = GROUP_CLEAN
}
