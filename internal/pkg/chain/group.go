package chain

import (
	"bytes"
	"encoding/hex"
	"errors"
	"time"

	"github.com/huo-ju/quorum/internal/pkg/nodectx"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
	"google.golang.org/protobuf/proto"
)

const (
	USER_CHANNEL_PREFIX     = "user_channel_"
	PRODUCER_CHANNEL_PREFIX = "prod_channel_"
)

type Group struct {
	//Group Item
	Item     *quorumpb.GroupItem
	ChainCtx *Chain
}

var group_log = logging.Logger("group")

func (grp *Group) Init(item *quorumpb.GroupItem) {
	group_log.Infof("Init called")
	grp.Item = item

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.Init(grp)
	grp.ChainCtx.LoadProducer()
}

//teardown group
func (grp *Group) Teardown() {
	group_log.Infof("Teardown called")

	if grp.ChainCtx.Syncer.Status == SYNCING_BACKWARD || grp.ChainCtx.Syncer.Status == SYNCING_FORWARD {
		grp.ChainCtx.Syncer.stopWaitBlock()
	}
}

func (grp *Group) CreateGrp(item *quorumpb.GroupItem) error {
	group_log.Infof("CreateGrp called")

	grp.Init(item)

	err := nodectx.GetDbMgr().AddGensisBlock(item.GenesisBlock, grp.ChainCtx.nodename)
	if err != nil {
		return err
	}

	group_log.Infof("Add owner as the first producer")
	//add owner as the first producer
	var pItem *quorumpb.ProducerItem
	pItem = &quorumpb.ProducerItem{}
	pItem.GroupId = item.GroupId
	pItem.GroupOwnerPubkey = item.OwnerPubKey
	pItem.ProducerPubkey = item.OwnerPubKey

	var buffer bytes.Buffer
	buffer.Write([]byte(pItem.GroupId))
	buffer.Write([]byte(pItem.ProducerPubkey))
	buffer.Write([]byte(pItem.GroupOwnerPubkey))
	hash := Hash(buffer.Bytes())

	ks := nodectx.GetNodeCtx().Keystore
	signature, err := ks.SignByKeyName(item.GroupId, hash)
	if err != nil {
		return err
	}

	pItem.GroupOwnerSign = hex.EncodeToString(signature)
	pItem.Memo = "Owner Registrate as the first oroducer"
	pItem.TimeStamp = time.Now().UnixNano()

	err = nodectx.GetDbMgr().AddProducer(pItem, grp.ChainCtx.nodename)
	if err != nil {
		return err
	}

	//reload producers
	grp.ChainCtx.LoadProducer()

	return nodectx.GetDbMgr().AddGroup(grp.Item)

}

func (grp *Group) DelGrp() error {
	group_log.Infof("DelGrp called")
	if grp.Item.UserSignPubkey != grp.Item.OwnerPubKey {
		err := errors.New("You can not 'delete' group created by others, use 'leave' instead")
		return err
	}

	err := grp.clearGroup()
	if err != nil {
		return err
	}

	return nodectx.GetDbMgr().RmGroup(grp.Item)
}

func (grp *Group) LeaveGrp() error {
	group_log.Infof("LeaveGrp called")
	if grp.Item.UserSignPubkey == grp.Item.OwnerPubKey {
		err := errors.New("Group creator can not leave the group they created, use 'delete' instead")
		return err
	}

	err := grp.clearGroup()
	if err != nil {
		return err
	}

	return nodectx.GetDbMgr().RmGroup(grp.Item)
}

func (grp *Group) clearGroup() error {

	//remove all group blocks (both cached and normal)

	//remove all group producers

	//remove all group trx

	//remove all group POST

	//remove all group CONTENT

	//remove all group Auth

	//remove all group Announce

	//remove all group schema

	return nil
}

func (grp *Group) StartSync() error {
	group_log.Infof("StartSync called")
	if grp.ChainCtx.Syncer.Status == SYNCING_BACKWARD || grp.ChainCtx.Syncer.Status == SYNCING_FORWARD {
		return errors.New("Group is syncing, don't start again")
	}

	for _, blockId := range grp.ChainCtx.group.Item.HighestBlockId {
		topBlock, err := nodectx.GetDbMgr().GetBlock(blockId, false, grp.ChainCtx.nodename)
		if err != nil {
			group_log.Warningf("Get top block error, blockId %s at %s, %s", blockId, grp.ChainCtx.nodename, err.Error())
			return err
		}
		return grp.ChainCtx.StartInitialSync(topBlock)
	}

	return nil
}

func (grp *Group) StopSync() error {
	group_log.Infof("StopSync called")
	if grp.ChainCtx.Syncer.Status == SYNCING_BACKWARD || grp.ChainCtx.Syncer.Status == SYNCING_FORWARD {
		grp.ChainCtx.StopSync()
		group_log.Infof("Sync stopped")
	}

	return nil
}

func (grp *Group) GetGroupCtn(filter string) ([]*quorumpb.PostItem, error) {
	group_log.Infof("GetGroupCtn called")
	return nodectx.GetDbMgr().GetGrpCtnt(grp.Item.GroupId, filter, grp.ChainCtx.nodename)
}

func (grp *Group) GetBlock(blockId string) (*quorumpb.Block, error) {
	group_log.Infof("GetBlock called")
	return nodectx.GetDbMgr().GetBlock(blockId, false, grp.ChainCtx.nodename)
}

func (grp *Group) GetTrx(trxId string) (*quorumpb.Trx, error) {
	group_log.Infof("GetTrx called")
	return nodectx.GetDbMgr().GetTrx(trxId, grp.ChainCtx.nodename)
}

func (grp *Group) GetBlockedUser() ([]*quorumpb.DenyUserItem, error) {
	group_log.Infof("GetBlockedUser called")
	return nodectx.GetDbMgr().GetBlkedUsers(grp.ChainCtx.nodename)
}

func (grp *Group) GetProducers() ([]*quorumpb.ProducerItem, error) {
	group_log.Infof("GetProducers called")
	return nodectx.GetDbMgr().GetProducers(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) UpdAnnounce(item *quorumpb.AnnounceItem) (string, error) {
	return grp.ChainCtx.Consensus.User().UpdAnnounce(item)
}

func (grp *Group) UpdBlkList(item *quorumpb.DenyUserItem) (string, error) {
	return grp.ChainCtx.Consensus.User().UpdBlkList(item)
}

func (grp *Group) PostToGroup(content proto.Message) (string, error) {
	return grp.ChainCtx.Consensus.User().PostToGroup(content)
}

func (grp *Group) UpdProducer(item *quorumpb.ProducerItem) (string, error) {
	return grp.ChainCtx.Consensus.User().UpdProducer(item)
}

func (grp *Group) UpdSchema(item *quorumpb.SchemaItem) (string, error) {
	return grp.ChainCtx.Consensus.User().UpdSchema(item)
}
