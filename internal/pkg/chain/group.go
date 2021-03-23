package chain

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/golang/glog"
	"time"
)

type GroupStatus int8

const (
	GROUP_SYNCING GroupStatus = 0 //syncing
	GROUP_OK      GroupStatus = 1 //normal
	GROUP_ERR     GroupStatus = 2 //error
)

type GroupUserItem struct {
	Uid       string
	PublicKey string
}

type GroupContentItem struct {
	Cid          string
	Publisher    string
	PublisherKey string
	Content      string
}

type GroupItem struct {
	OwnerPubKey string
	IsDirty     bool
	GroupStatus GroupStatus
	LastUpdate  uint64

	GenesisBlock Block

	LatestBlockNum int64
	LatestBlockId  string

	ContentDb  *badger.DB //Content db
	BlockSeqDb *badger.DB //map blocknum with blockId
	UserDb     *badger.DB //Users db

	ContentsList []string
	UsersMap     map[string]GroupUserItem //Group Users   (index map)

	//private pubsub channel
	//PubSubTopic libp2p.pubsub

	AskNextTicker *time.Ticker
	TickerDone    chan bool
}

func (item *GroupItem) AddBlock(block Block) error {

	//verify block
	if block.BlockNum != 0 {
		var topBlock Block
		topBlock, _ = item.GetTopBlock()
		valid, _ := IsBlockValid(block, topBlock)
		if !valid {
			glog.Errorf("Invalid block")
			//throw error, invalid block
			return nil
		}
	}

	//save block to local db
	AddBlock(block, item)

	//update BlockSeqDb
	err := item.BlockSeqDb.Update(func(txn *badger.Txn) error {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(block.BlockNum))
		e := badger.NewEntry(b, []byte(block.Cid))
		err := txn.SetEntry(e)
		return err
	})

	if err != nil {
		glog.Fatalf(err.Error())
	}

	item.LatestBlockNum = block.BlockNum
	item.LatestBlockId = block.Cid

	//update contentDB
	for _, v := range block.Trxs {
		item.ContentsList = append(item.ContentsList, string(v.Data))
	}

	fmt.Printf("-- add new block, id::%s, num::%d --\n", block.Cid, block.BlockNum)

	glog.Infof("<<<<<<<<<<<<<<<<< content list >>>>>>>>>>>>>>>>>>>")
	for _, c := range item.ContentsList {
		glog.Infof(c)
	}

	return nil
}

func (item *GroupItem) GetTopBlock() (Block, error) {
	topBlock, err := GetBlock(item.LatestBlockId)
	return topBlock, err
}

func (item *GroupItem) GetBlockIdByBlockNum(blockNum int64) (string, error) {
	var blockId string
	err := item.BlockSeqDb.View(func(txn *badger.Txn) error {
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

//group protocols

//Add new Group
func (item *GroupItem) AddNewGroup() error {
	return nil
}

//Rm group
func (item *GroupItem) RmGroup() error {
	return nil
}

//Add Group User
func (item *GroupItem) AddGroupUser() error {
	return nil
}

//Rm Group User
func (item *GroupItem) RmGroupUser() error {
	return nil
}

//Add Content to Group
func (item *GroupItem) AddGroupContent() error {
	return nil
}

func (item *GroupItem) RmGroupContent() error {
	return nil
}

//Sync in memory content map with local DB
func (item *GroupItem) syncContent() error {
	return nil
}

//Sync in memory user map with local db
func (item *GroupItem) syncUser() error {
	return nil
}

//Load groupItem from DB
func (item *GroupItem) LoadContent(upperCount, lowerCount uint64) error {
	return nil
}

//Load Group User from DB
func (item *GroupItem) LoadUser() error {
	return nil
}

//Release in memory group Item
func (item *GroupItem) ReleaseContent() error {
	return nil
}

//Release Group User
func (item *GroupItem) ReleaseUser() error {
	return nil
}

//test only
const TestGroupId string = "test_group_id"

func JoinTestGroup() error {

	glog.Infof("<<<Join test group>>>")

	var group *GroupItem
	group = &GroupItem{}

	group.OwnerPubKey = "owner_pub_key"
	group.IsDirty = true
	group.GroupStatus = GROUP_OK
	group.LastUpdate = 0

	var err error
	group.ContentDb, err = badger.Open(badger.DefaultOptions(GetContext().DataPath + TestGroupId + "/" + "_group"))
	group.UserDb, err = badger.Open(badger.DefaultOptions(GetContext().DataPath + TestGroupId + "/" + "_user"))
	group.BlockSeqDb, err = badger.Open(badger.DefaultOptions(GetContext().DataPath + TestGroupId + "/" + "_bsq"))

	if err != nil {
		glog.Fatal(err.Error())
	}

	//defer group.ContentDb.Close()
	//defer group.UserDb.Close()
	//defer group.BlockSeqDb.Close()

	var genesisBlock Block
	genesisBlock.Cid = "12345678"
	genesisBlock.GroupId = TestGroupId
	genesisBlock.PrevBlockId = ""
	genesisBlock.PreviousHash = ""
	genesisBlock.BlockNum = 0
	genesisBlock.Timestamp = 0 //test_time_stamp

	group.GenesisBlock = genesisBlock
	GetContext().Groups[TestGroupId] = group

	AddBlock(genesisBlock, group)
	group.AddBlock(genesisBlock)

	if group.IsDirty {
		group.StartAskNextBlock()
	}
	return nil
}

func (item *GroupItem) StartAskNextBlock() {
	//send ask_next_block every 3 sec till get "on_top response"
	item.AskNextTicker = time.NewTicker(1000 * time.Millisecond)
	item.TickerDone = make(chan bool)
	go func() {
		for {
			select {
			case <-item.TickerDone:
				return
			case t := <-item.AskNextTicker.C:
				glog.Infof("Ask NEXT_BLOCK " + t.UTC().String())
				//send ask next block msg out
				topBlock, _ := item.GetTopBlock()
				askNextMsg, _ := CreateTrxReqNextBlock(topBlock)
				jsonBytes, _ := json.Marshal(askNextMsg)
				GetContext().PublicTopic.Publish(GetContext().Ctx, jsonBytes)
			}
		}
	}()

	//return nil
}

func (item *GroupItem) StopAskNextBlock() {
	item.AskNextTicker.Stop()
	item.TickerDone <- true
	item.IsDirty = false
}
