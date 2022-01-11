package chain

import (
	"bytes"
	"encoding/hex"
	"errors"

	//"fmt"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

const (
	USER_CHANNEL_PREFIX     = "user_channel_"
	PRODUCER_CHANNEL_PREFIX = "prod_channel_"
	SYNC_CHANNEL_PREFIX     = "sync_channel_"
)

type Group struct {
	//Group Item
	Item     *quorumpb.GroupItem
	ChainCtx *Chain
}

var group_log = logging.Logger("group")

func (grp *Group) Init(item *quorumpb.GroupItem) {
	group_log.Debugf("<%s> Init called", item.GroupId)
	grp.Item = item

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.Init(grp)

	//reload producers
	grp.ChainCtx.UpdProducerList()
	grp.ChainCtx.CreateConsensus()

	nodectx.GetNodeCtx().Node.RumExchange.ChainReg(item.GroupId, grp.ChainCtx)
	group_log.Infof("Group <%s> initialed", grp.Item.GroupId)
}

//teardown group
func (grp *Group) Teardown() {
	group_log.Debugf("<%s> Teardown called", grp.Item.GroupId)

	if grp.ChainCtx.Syncer.Status == SYNCING_BACKWARD || grp.ChainCtx.Syncer.Status == SYNCING_FORWARD {
		grp.ChainCtx.Syncer.stopWaitBlock()
	}

	group_log.Infof("Group <%s> teardown", grp.Item.GroupId)
}

func (grp *Group) CreateGrp(item *quorumpb.GroupItem) error {
	group_log.Debugf("<%s> CreateGrp called", item.GroupId)

	grp.Item = item

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.Init(grp)

	err := nodectx.GetDbMgr().AddGensisBlock(item.GenesisBlock, grp.ChainCtx.nodename)
	if err != nil {
		return err
	}

	group_log.Debugf("<%s> add owner as the first producer", grp.Item.GroupId)
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
	pItem.Memo = "Owner Registated as the first oroducer"
	pItem.TimeStamp = time.Now().UnixNano()

	err = nodectx.GetDbMgr().AddProducer(pItem, grp.ChainCtx.nodename)
	if err != nil {
		return err
	}

	group_log.Infof("Group <%s> created", grp.Item.GroupId)

	err = nodectx.GetDbMgr().AddGroup(grp.Item)
	if err != nil {
		return err
	}

	//reload producers
	grp.ChainCtx.UpdProducerList()
	grp.ChainCtx.CreateConsensus()

	return nil
}

func (grp *Group) LeaveGrp() error {
	group_log.Debugf("<%s> LeaveGrp called", grp.Item.GroupId)

	grp.StopSync()
	//leave pubsub channel
	grp.ChainCtx.LeaveChannel()
	group_log.Infof("Group <%s> leaved", grp.Item.GroupId)
	return nodectx.GetDbMgr().RmGroup(grp.Item)
}

func (grp *Group) ClearGroup() error {
	return nodectx.GetDbMgr().RemoveGroupData(grp.Item, grp.ChainCtx.nodename)
}

func (grp *Group) StartSync() error {
	group_log.Debugf("<%s> StartSync called", grp.Item.GroupId)
	if grp.ChainCtx.Syncer.Status == SYNCING_BACKWARD || grp.ChainCtx.Syncer.Status == SYNCING_FORWARD {
		return errors.New("Group is syncing, don't start again")
	}

	higestBId := grp.ChainCtx.group.Item.HighestBlockId
	topBlock, err := nodectx.GetDbMgr().GetBlock(higestBId, false, grp.ChainCtx.nodename)
	if err != nil {
		group_log.Warningf("Get top block error, blockId <%s> at <%s>, <%s>", higestBId, grp.ChainCtx.nodename, err.Error())
		return err
	}

	group_log.Infof("Group <%s> start sync", grp.Item.GroupId)
	return grp.ChainCtx.StartInitialSync(topBlock)
}

func (grp *Group) StopSync() error {
	group_log.Debugf("<%s> StopSync called", grp.Item.GroupId)
	if grp.ChainCtx.Syncer.Status == SYNCING_BACKWARD || grp.ChainCtx.Syncer.Status == SYNCING_FORWARD {
		grp.ChainCtx.StopSync()
	}

	group_log.Infof("Group <%s> stop sync", grp.Item.GroupId)
	return nil
}

func (grp *Group) GetGroupCtn(filter string) ([]*quorumpb.PostItem, error) {
	group_log.Debugf("<%s> GetGroupCtn called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetGrpCtnt(grp.Item.GroupId, filter, grp.ChainCtx.nodename)
}

func (grp *Group) GetBlock(blockId string) (*quorumpb.Block, error) {
	group_log.Debugf("<%s> GetBlock called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetBlock(blockId, false, grp.ChainCtx.nodename)
}

func (grp *Group) GetTrx(trxId string) (*quorumpb.Trx, error) {
	group_log.Debugf("<%s> GetTrx called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetTrx(trxId, grp.ChainCtx.nodename)
}

func (grp *Group) GetBlockedUser() ([]*quorumpb.DenyUserItem, error) {
	group_log.Debugf("<%s> GetBlockedUser called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetBlkedUsers(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetProducers() ([]*quorumpb.ProducerItem, error) {
	group_log.Debugf("<%s> GetProducers called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetProducers(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetSchemas() ([]*quorumpb.SchemaItem, error) {
	group_log.Debugf("<%s> GetSchema called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAllSchemasByGroup(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetAnnouncedProducers() ([]*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedProducer called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAnnounceProducersByGroup(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetAnnouncedUsers() ([]*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedUser called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAnnounceUsersByGroup(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetAnnouncedProducer(pubkey string) (*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedProducer called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAnnouncedProducer(grp.Item.GroupId, pubkey, grp.ChainCtx.nodename)
}

func (grp *Group) GetAnnouncedUser(pubkey string) (*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedUser called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAnnouncedUser(grp.Item.GroupId, pubkey, grp.ChainCtx.nodename)
}

func (grp *Group) GetGroupConfigKeyList() (keyName []string, itemType []string, err error) {
	group_log.Debugf("<%s> GetGroupConfigKeyList called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetGroupConfigKey(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetGroupConfigItem(keyName string) (*quorumpb.GroupConfigItem, error) {
	group_log.Debugf("<%s> GetGroupConfigKeyList called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetGroupConfigItem(keyName, grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetGroupConfigItemBool(keyName string) (bool, error) {
	group_log.Debugf("<%s> GetGroupConfigItemBool called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetGroupConfigItemBool(keyName, grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetGroupConfigItemInt(keyName string) (int, error) {
	group_log.Debugf("<%s> GetGroupConfigItemInt called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetGroupConfigItemInt(keyName, grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetGroupConfigItemString(keyName string) (string, error) {
	group_log.Debugf("<%s> GetGroupConfigItemString called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetGroupConfigItemString(keyName, grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) UpdAnnounce(item *quorumpb.AnnounceItem) (string, error) {
	group_log.Debugf("<%s> UpdAnnounce called", grp.Item.GroupId)
	return grp.ChainCtx.Consensus.User().UpdAnnounce(item)
}

func (grp *Group) UpdBlkList(item *quorumpb.DenyUserItem) (string, error) {
	group_log.Debugf("<%s> UpdBlkList called", grp.Item.GroupId)
	return grp.ChainCtx.Consensus.User().UpdBlkList(item)
}

func (grp *Group) PostToGroup(content proto.Message) (string, error) {
	group_log.Debugf("<%s> PostToGroup called", grp.Item.GroupId)
	//if private group?
	if grp.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		keys, err := grp.ChainCtx.GetUsesEncryptPubKeys()
		if err != nil {
			return "", err
		}
		return grp.ChainCtx.Consensus.User().PostToGroup(content, keys)
	}
	return grp.ChainCtx.Consensus.User().PostToGroup(content)

}

func (grp *Group) UpdProducer(item *quorumpb.ProducerItem) (string, error) {
	group_log.Debugf("<%s> UpdProducer called", grp.Item.GroupId)
	return grp.ChainCtx.Consensus.User().UpdProducer(item)
}

func (grp *Group) UpdUser(item *quorumpb.UserItem) (string, error) {
	group_log.Debugf("<%s> UpdUser called", grp.Item.GroupId)
	return grp.ChainCtx.Consensus.User().UpdUser(item)
}

func (grp *Group) UpdSchema(item *quorumpb.SchemaItem) (string, error) {
	group_log.Debugf("<%s> UpdSchema called", grp.Item.GroupId)
	return grp.ChainCtx.Consensus.User().UpdSchema(item)
}

func (grp *Group) UpdGroupConfig(item *quorumpb.GroupConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdGroupConfig called", grp.Item.GroupId)
	return grp.ChainCtx.Consensus.User().UpdGroupConfig(item)
}

func (grp *Group) IsProducerAnnounced(producerSignPubkey string) (bool, error) {
	group_log.Debugf("<%s> IsProducerAnnounced called", grp.Item.GroupId)
	return nodectx.GetDbMgr().IsProducerAnnounced(grp.Item.GroupId, producerSignPubkey, grp.ChainCtx.nodename)
}

func (grp *Group) IsUserAnnounced(userSignPubkey string) (bool, error) {
	group_log.Debugf("<%s> IsUserAnnounced called", grp.Item.GroupId)
	return nodectx.GetDbMgr().IsUserAnnounced(grp.Item.GroupId, userSignPubkey, grp.ChainCtx.nodename)
}
