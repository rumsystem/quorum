package nodesdkdb

import (
	"errors"
	"sync"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var dbmgr_log = logging.Logger("nodesdk_dbmgr")

const NODESDK_PREFIX string = "nodesdk_" //nodesdk group
const NONCE_PREFIX string = "nonce_"     //group trx nonce

//groupinfo db
const GROUPITEM_PREFIX string = "grpitem_" //relay

type DbMgr struct {
	GroupInfoDb QuorumStorage
	Db          QuorumStorage
	seq         sync.Map
	DataPath    string
}

func (dbMgr *DbMgr) CloseDb() {
	dbMgr.GroupInfoDb.Close()
	dbMgr.Db.Close()
	dbmgr_log.Infof("ChainCtx Db closed")
}

func (dbMgr *DbMgr) AddGroup(groupItem *quorumpb.NodeSDKGroupItem) error {
	//check if group exist
	key := NODESDK_PREFIX + GROUPITEM_PREFIX + groupItem.Group.GroupId
	exist, err := dbMgr.GroupInfoDb.IsExist([]byte(key))
	if exist {
		return errors.New("Group with same GroupId existed")
	}

	//add group to db
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}
	return dbMgr.GroupInfoDb.Set([]byte(key), value)
}

func (dbMgr *DbMgr) UpdGroup(groupItem *quorumpb.NodeSDKGroupItem) error {
	value, err := proto.Marshal(groupItem)
	if err != nil {
		return err
	}

	key := NODESDK_PREFIX + GROUPITEM_PREFIX + groupItem.Group.GroupId
	exist, err := dbMgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		return errors.New("Group is not existed")
	}

	//upd group to db
	return dbMgr.GroupInfoDb.Set([]byte(key), value)
}

func (dbMgr *DbMgr) RmGroup(groupId string) error {
	//check if group exist
	key := NODESDK_PREFIX + GROUPITEM_PREFIX + groupId
	exist, err := dbMgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return err
		}
		return errors.New("Group Not Found")
	}

	//delete group
	return dbMgr.GroupInfoDb.Delete([]byte(key))
}

//Get group list
func (dbMgr *DbMgr) GetGroupsBytes() ([][]byte, error) {
	var groupItemList [][]byte
	key := NODESDK_PREFIX + GROUPITEM_PREFIX
	err := dbMgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		groupItemList = append(groupItemList, v)
		return nil
	})
	return groupItemList, err
}

//Get Gorup Info
func (dbMgr *DbMgr) GetGroupInfo(groupId string) (*quorumpb.NodeSDKGroupItem, error) {
	key := NODESDK_PREFIX + GROUPITEM_PREFIX + groupId
	exist, err := dbMgr.GroupInfoDb.IsExist([]byte(key))
	if !exist {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("Group Not Found")
	}

	groupInfoByte, err := dbMgr.GroupInfoDb.Get([]byte(key))
	if err != nil {
		return nil, err
	}

	var groupInfo *quorumpb.NodeSDKGroupItem
	groupInfo = &quorumpb.NodeSDKGroupItem{}
	err = proto.Unmarshal(groupInfoByte, groupInfo)
	if err != nil {
		return nil, err
	}

	//delete group
	return groupInfo, nil
}

func (dbMgr *DbMgr) GetAllGroups() ([]*quorumpb.NodeSDKGroupItem, error) {
	var result []*quorumpb.NodeSDKGroupItem

	key := NODESDK_PREFIX + GROUPITEM_PREFIX
	err := dbMgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		var item *quorumpb.NodeSDKGroupItem
		item = &quorumpb.NodeSDKGroupItem{}
		err = proto.Unmarshal(v, item)
		if err != nil {
			return err
		}
		result = append(result, item)
		return nil
	})
	return result, err
}

//get next nonce
func (dbMgr *DbMgr) GetNextNouce(groupId string, prefix ...string) (uint64, error) {
	key := NONCE_PREFIX + "_" + groupId

	nonceseq, succ := dbMgr.seq.Load(key)
	if succ == false {
		newseq, err := dbMgr.Db.GetSequence([]byte(key), 1)
		if err != nil {
			return 0, err
		}
		dbMgr.seq.Store(key, newseq)
		return newseq.Next()
	} else {
		return nonceseq.(Sequence).Next()
	}
}

/*
	//test only, show db contents
	err = dbMgr.TrxDb.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			err := item.Value(func(v []byte) error {
				fmt.Printf("key=%s, value=%s\n", k, v)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})*/
