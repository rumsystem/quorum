package chainstorage

import (
	"errors"
	"time"

	guuid "github.com/google/uuid"
	s "github.com/rumsystem/quorum/internal/pkg/storage"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

// relaystatus: req, approved and activity
func (cs *Storage) AddRelayReq(groupRelayItem *quorumpb.GroupRelayItem) (string, error) {
	groupRelayItem.RelayId = guuid.New().String()

	//dbMgr.GroupInfoDb.PrefixDelete([]byte(RELAY_PREFIX))

	key := s.GetRelayReqKey(groupRelayItem.GroupId, groupRelayItem.Type)
	if groupRelayItem.Type == "user" {
		key = s.GetRelayReqUserKey(groupRelayItem.GroupId, groupRelayItem.Type, groupRelayItem.UserPubkey)
	}

	//check if group relay req exist
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if exist { //check if not expire
		return "", errors.New("the same relay req exist ")
	}

	//add group relay req to db
	value, err := proto.Marshal(groupRelayItem)
	if err != nil {
		return "", err
	}
	return groupRelayItem.RelayId, cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) AddRelayActivity(groupRelayItem *quorumpb.GroupRelayItem) (string, error) {
	key := s.GetRelayActivityKey(groupRelayItem.GroupId, groupRelayItem.Type)
	//check if group relay req exist
	exist, err := cs.dbmgr.GroupInfoDb.IsExist([]byte(key))
	if exist { //check if not expire
		return "", errors.New("the same relay exist ")
	}

	//add group relay to db
	value, err := proto.Marshal(groupRelayItem)
	if err != nil {
		return "", err
	}
	return groupRelayItem.RelayId, cs.dbmgr.GroupInfoDb.Set([]byte(key), value)
}

func (cs *Storage) DeleteRelay(relayid string) (bool, *quorumpb.GroupRelayItem, error) {
	key := s.GetRelayPrefix()
	succ := false
	relayitem := quorumpb.GroupRelayItem{}
	err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		err = proto.Unmarshal(v, &relayitem)
		if err == nil {
			if relayitem.RelayId == relayid {
				err = cs.dbmgr.GroupInfoDb.Delete(k)
				if err == nil {
					succ = true
				}
			}
		}
		return nil
	})
	return succ, &relayitem, err
}

func (cs *Storage) ApproveRelayReq(reqid string) (bool, *quorumpb.GroupRelayItem, error) {
	key := s.GetRelayReqPrefix()
	succ := false

	relayreq := quorumpb.GroupRelayItem{}
	err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
		if err != nil {
			return err
		}
		err = proto.Unmarshal(v, &relayreq)
		if relayreq.RelayId == reqid {
			relayreq.ApproveTime = time.Now().UnixNano()
			approvedkey := s.GetRelayApprovedKey(relayreq.GroupId, relayreq.Type)
			approvedvalue, err := proto.Marshal(&relayreq)
			if err != nil {
				return err
			}
			err = cs.dbmgr.GroupInfoDb.Set([]byte(approvedkey), approvedvalue)
			if err != nil {
				return err
			}
			succ = true
			return cs.dbmgr.GroupInfoDb.Delete(k)
		}
		return nil
	})
	return succ, &relayreq, err
}

func (cs *Storage) GetRelay(relaystatus string, groupid string) ([]*quorumpb.GroupRelayItem, error) {
	switch relaystatus {
	case "req", "approved", "activity":
		key := s.RELAY_PREFIX + "_" + relaystatus + "_" + groupid
		groupRelayItemList := []*quorumpb.GroupRelayItem{}
		err := cs.dbmgr.GroupInfoDb.PrefixForeach([]byte(key), func(k []byte, v []byte, err error) error {
			if err != nil {
				return err
			}
			relayreq := quorumpb.GroupRelayItem{}
			err = proto.Unmarshal(v, &relayreq)
			groupRelayItemList = append(groupRelayItemList, &relayreq)
			return nil
		})
		return groupRelayItemList, err
	}
	return nil, errors.New("unknown relaystatus")
}

func (cs *Storage) GetRelayReq(groupid string) ([]*quorumpb.GroupRelayItem, error) {
	return cs.GetRelay("req", groupid)
}

func (cs *Storage) GetRelayApproved(groupid string) ([]*quorumpb.GroupRelayItem, error) {
	return cs.GetRelay("approved", groupid)
}

func (cs *Storage) GetRelayActivity(groupid string) ([]*quorumpb.GroupRelayItem, error) {
	return cs.GetRelay("activity", groupid)
}
