package chain

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	//"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/golang/glog"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"time"
)

type GroupStatus int8

const (
	GROUP_CLEAN = 0
	GROUP_DIRTY = 1
)

type GroupContentItem struct {
	TrxId     string
	Publisher string
	Content   string
	TimeStamp int64
}

type Group struct {
	Item          *GroupItem
	Db            *GroupDb
	AskNextTicker *time.Ticker
	TickerDone    chan bool
	Status        GroupStatus
}

type GroupItem struct {
	OwnerPubKey    string
	GroupId        string
	GroupName      string
	LastUpdate     int64
	LatestBlockNum int64
	LatestBlockId  string

	GenesisBlock Block
}

type GroupDb struct {
	ContentDb  *badger.DB //Content db
	BlockSeqDb *badger.DB //map blocknum with blockId
}

func (grp *Group) init(item *GroupItem) error {
	grp.Item = item

	var db *GroupDb
	db = &GroupDb{}

	contentDb, err := badger.Open(badger.DefaultOptions(GetDbMgr().DataPath + item.GroupId + "/" + "_group"))
	if err != nil {
		return err
	}

	blockSeqDb, err := badger.Open(badger.DefaultOptions(GetDbMgr().DataPath + item.GroupId + "/" + "_bsq"))
	if err != nil {
		return err
	}

	db.ContentDb = contentDb
	db.BlockSeqDb = blockSeqDb
	grp.Db = db

	grp.AskNextTicker = time.NewTicker(1000 * time.Millisecond)
	grp.TickerDone = make(chan bool)

	return nil
}

func (grp *Group) Teardown() {
	//is syncing, stop ask next task
	if grp.Status == GROUP_DIRTY {
		grp.stopAskNextBlock()
	}
	grp.Db.ContentDb.Close()
	grp.Db.BlockSeqDb.Close()
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

func (grp *Group) AddBlock(block Block) error {
	//verify block
	if block.BlockNum != 0 {
		var topBlock Block
		topBlock, _ = grp.GetTopBlock()
		valid, _ := IsBlockValid(block, topBlock)
		if !valid {
			glog.Errorf("Invalid block")
			//throw error, invalid block
			return nil
		}
	}

	//save block to local db
	GetDbMgr().AddBlock(block)

	//update BlockSeqDb
	err := grp.Db.BlockSeqDb.Update(func(txn *badger.Txn) error {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(block.BlockNum))
		e := badger.NewEntry(b, []byte(block.Cid))
		err := txn.SetEntry(e)
		return err
	})

	if err != nil {
		return err
	}

	//update contentDB
	for _, trx := range block.Trxs {

		var ctnItem *GroupContentItem
		ctnItem = &GroupContentItem{}

		ctnItem.TrxId = trx.Msg.TrxId
		ctnItem.Publisher = trx.Msg.Sender
		ctnItem.Content = string(trx.Data)
		ctnItem.TimeStamp = trx.Msg.TimeStamp
		ctnBytes, err := json.Marshal(ctnItem)
		if err != nil {
			return err
		}

		//update ContentDb
		err = grp.Db.ContentDb.Update(func(txn *badger.Txn) error {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, uint64(trx.Msg.TimeStamp))
			e := badger.NewEntry(b, ctnBytes)
			err := txn.SetEntry(e)
			return err
		})

		if err != nil {
			return err
		}
	}

	grp.Item.LatestBlockNum = block.BlockNum
	grp.Item.LatestBlockId = block.Cid
	grp.Item.LastUpdate = time.Now().UnixNano()

	//update local db
	dbMgr.UpdGroup(grp.Item)

	return nil
}

func (grp *Group) GetTopBlock() (Block, error) {
	topBlock, err := GetDbMgr().GetBlock(grp.Item.LatestBlockId)
	return topBlock, err
}

func (grp *Group) GetBlockIdByBlockNum(blockNum int64) (string, error) {
	var blockId string
	err := grp.Db.BlockSeqDb.View(func(txn *badger.Txn) error {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(blockNum))
		item, err := txn.Get([]byte(b))

		if err != nil {
			return err
		}

		blockIdBytes, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		blockId = string(blockIdBytes)
		return nil
	})
	return blockId, err
}

func (grp *Group) CreateGrp(item *GroupItem) error {
	err := grp.init(item)
	if err != nil {
		return err
	}

	err = dbMgr.AddBlock(item.GenesisBlock)
	if err != nil {
		return err
	}
	return dbMgr.AddGroup(grp.Item)
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

	return dbMgr.RmGroup(grp.Item)
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

	return dbMgr.RmGroup(grp.Item)
}

//Add Content to Group
func (grp *Group) Post(content string) (string, error) {

	var trx Trx
	var trxMsg TrxMsg

	//use test groupId here, should parse from POST msg
	trxMsg, _ = CreateTrxMsgReqSign(grp.Item.GroupId, []byte(content))
	trx.Msg = trxMsg
	trx.Data = []byte(content)
	var cons []string
	trx.Consensus = cons

	dbMgr.AddTrx(trx)

	jsonBytes, err := json.Marshal(trxMsg)
	if err != nil {
		return "INVALID_TRX", err
	}

	chainCtx.PublicTopic.Publish(chainCtx.Ctx, jsonBytes)
	return trxMsg.TrxId, nil
}

//Load groupItem from DB
func (grp *Group) GetContent(upperCount, lowerCount uint64) []*GroupContentItem {
	return nil
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

				jsonBytes, err := json.Marshal(askNextMsg)
				if err != nil {
					glog.Fatalf(err.Error())
				}

				GetChainCtx().PublicTopic.Publish(GetChainCtx().Ctx, jsonBytes)
			}
		}
	}()

	//return nil
}

func (grp *Group) stopAskNextBlock() {
	grp.AskNextTicker.Stop()
	grp.TickerDone <- true
	grp.Status = GROUP_CLEAN
}
