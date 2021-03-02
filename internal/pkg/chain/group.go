package chain

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/golang/glog"
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
	//should protected by mutex {
	IsDirty     bool
	GroupStatus GroupStatus
	LastUpdate  string
	//}

	GenesisBlock Block

	ContentsList []GroupContentItem //Group Contents (in time sequence)
	BlocksList   []Block            //Group Block (in time sequence)

	ContentMap map[string]GroupContentItem //Group Content (index map)
	UsersMap   map[string]GroupUserItem    //Group Users   (index map)
	BlocksMap  map[string]Block            //Group Blocks  (index map)

	ContentDb *badger.DB //Content db
	UildCidDb *badger.DB //map blockId with cid
	UserDb    *badger.DB //Users db

	//private pubsub channel
	//PubSubTopic libp2p.pubsub
}

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

	var contentList []GroupContentItem
	var blocksList []Block

	var contentMap map[string]GroupContentItem
	contentMap = make(map[string]GroupContentItem)

	var usersMap map[string]GroupUserItem
	usersMap = make(map[string]GroupUserItem)

	var blocksMap map[string]Block
	blocksMap = make(map[string]Block)

	genesisBlock := CreateGenesisBlock()

	//create test group
	var group GroupItem

	group.OwnerPubKey = "owner_pub_key"
	group.IsDirty = false
	group.GroupStatus = GROUP_OK
	group.LastUpdate = "last_update"

	group.GenesisBlock = genesisBlock

	group.ContentsList = contentList
	group.BlocksList = blocksList

	group.ContentMap = contentMap
	group.UsersMap = usersMap
	group.BlocksMap = blocksMap

	group.ContentDb = nil
	group.UildCidDb = nil
	group.UserDb = nil

	group.BlocksList = append(group.BlocksList, genesisBlock)

	chainContext.Group[TestGroupId] = group
	return nil
}
