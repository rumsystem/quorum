package chain

import (
	"bytes"
	"encoding/hex"

	"time"

	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
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

	//register chainctx with conn
	conn.GetConn().RegisterChainCtx(item.GroupId, item.OwnerPubKey, item.UserSignPubkey, grp.ChainCtx)

	//reload producers
	grp.ChainCtx.UpdProducerList()

	//reload all announced user(if private)
	if grp.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		group_log.Debugf("<%s> Private group load announced user key", item.GroupId)
		grp.ChainCtx.UpdUserList()
	}

	grp.ChainCtx.CreateConsensus()

	//start send snapshot
	grp.ChainCtx.StartSnapshot()

	group_log.Infof("Group <%s> initialed", grp.Item.GroupId)
}

func (grp *Group) SetRumExchangeTestMode() {
	grp.ChainCtx.SetRumExchangeTestMode()
}

//teardown group
func (grp *Group) Teardown() {
	group_log.Debugf("<%s> Teardown called", grp.Item.GroupId)

	//unregisted chainctx with conn
	conn.GetConn().UnregisterChainCtx(grp.Item.GroupId)

	//stop snapshot
	grp.ChainCtx.StopSnapshot()

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

	group_log.Debugf("<%s> Update nonce called, with nodename <%s>", item.GroupId, grp.ChainCtx.nodename)
	//update nonce, set nonce to 0
	_, err = nodectx.GetDbMgr().UpdateNonce(item.GroupId, grp.ChainCtx.nodename)
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

	conn.GetConn().RegisterChainCtx(item.GroupId, item.OwnerPubKey, item.UserSignPubkey, grp.ChainCtx)

	//reload producers
	grp.ChainCtx.UpdProducerList()
	grp.ChainCtx.CreateConsensus()

	//start send snapshot
	grp.ChainCtx.StartSnapshot()

	return nil
}

func (grp *Group) LeaveGrp() error {
	group_log.Debugf("<%s> LeaveGrp called", grp.Item.GroupId)
	conn.GetConn().UnregisterChainCtx(grp.Item.GroupId)
	return nodectx.GetDbMgr().RmGroup(grp.Item)
}

func (grp *Group) ClearGroup() error {
	return nodectx.GetDbMgr().RemoveGroupData(grp.Item, grp.ChainCtx.nodename)
}

func (grp *Group) StartSync() error {
	group_log.Debugf("<%s> StartSync called", grp.Item.GroupId)
	return grp.ChainCtx.SyncForward(grp.ChainCtx.group.Item.HighestBlockId, grp.ChainCtx.nodename)
}

func (grp *Group) StopSync() error {
	group_log.Debugf("<%s> StopSync called", grp.Item.GroupId)
	return grp.ChainCtx.StopSync()
}

func (grp *Group) GetSyncerStatus() int8 {
	return grp.ChainCtx.syncer.Status
}

func (grp *Group) GetSnapshotInfo() (tag *quorumpb.SnapShotTag, err error) {
	return grp.ChainCtx.GetSnapshotTag()
}

func (grp *Group) GetGroupCtn(filter string) ([]*quorumpb.PostItem, error) {
	group_log.Debugf("<%s> GetGroupCtn called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetGrpCtnt(grp.Item.GroupId, filter, grp.ChainCtx.nodename)
}

func (grp *Group) GetBlock(blockId string) (*quorumpb.Block, error) {
	group_log.Debugf("<%s> GetBlock called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetBlock(blockId, false, grp.ChainCtx.nodename)
}

func (grp *Group) GetTrx(trxId string) (*quorumpb.Trx, []int64, error) {
	group_log.Debugf("<%s> GetTrx called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetTrx(trxId, storage.Chain, grp.ChainCtx.nodename)
}

func (grp *Group) GetTrxFromCache(trxId string) (*quorumpb.Trx, []int64, error) {
	group_log.Debugf("<%s> GetTrx called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetTrx(trxId, storage.Cache, grp.ChainCtx.nodename)
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
	group_log.Debugf("<%s> GetAnnouncedUsers called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAnnounceUsersByGroup(grp.Item.GroupId, grp.ChainCtx.nodename)
}

//func (grp *Group) GetAnnounceUser() ([]*quorumpb.AnnounceItem, error) {
//	group_log.Debugf("<%s> GetAnnouncedUser called", grp.Item.GroupId)
//	return nodectx.GetDbMgr().GetAnnounceUsersByGroup(grp.Item.GroupId, grp.ChainCtx.nodename)
//}

func (grp *Group) GetAnnouncedProducer(pubkey string) (*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedProducer called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAnnouncedProducer(grp.Item.GroupId, pubkey, grp.ChainCtx.nodename)
}

func (grp *Group) GetAnnouncedUser(pubkey string) (*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedUser called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAnnouncedUser(grp.Item.GroupId, pubkey, grp.ChainCtx.nodename)
}

func (grp *Group) GetAppConfigKeyList() (keyName []string, itemType []string, err error) {
	group_log.Debugf("<%s> GetAppConfigKeyList called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAppConfigKey(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetAppConfigItem(keyName string) (*quorumpb.AppConfigItem, error) {
	group_log.Debugf("<%s> GetAppConfigItem called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAppConfigItem(keyName, grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetAppConfigItemBool(keyName string) (bool, error) {
	group_log.Debugf("<%s> GetAppConfigItemBool called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAppConfigItemBool(keyName, grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetAppConfigItemInt(keyName string) (int, error) {
	group_log.Debugf("<%s> GetAppConfigItemInt called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAppConfigItemInt(keyName, grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetAppConfigItemString(keyName string) (string, error) {
	group_log.Debugf("<%s> GetAppConfigItemString called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetAppConfigItemString(keyName, grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) UpdAnnounce(item *quorumpb.AnnounceItem) (string, error) {
	group_log.Debugf("<%s> UpdAnnounce called", grp.Item.GroupId)
	return grp.ChainCtx.Consensus.User().UpdAnnounce(item)
}

func (grp *Group) PostToGroup(content proto.Message) (string, error) {
	group_log.Debugf("<%s> PostToGroup called", grp.Item.GroupId)
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

func (grp *Group) UpdAppConfig(item *quorumpb.AppConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdAppConfig called", grp.Item.GroupId)
	return grp.ChainCtx.Consensus.User().UpdAppConfig(item)
}

func (grp *Group) IsProducerAnnounced(producerSignPubkey string) (bool, error) {
	group_log.Debugf("<%s> IsProducerAnnounced called", grp.Item.GroupId)
	return nodectx.GetDbMgr().IsProducerAnnounced(grp.Item.GroupId, producerSignPubkey, grp.ChainCtx.nodename)
}

func (grp *Group) IsUserAnnounced(userSignPubkey string) (bool, error) {
	group_log.Debugf("<%s> IsUserAnnounced called", grp.Item.GroupId)
	return nodectx.GetDbMgr().IsUserAnnounced(grp.Item.GroupId, userSignPubkey, grp.ChainCtx.nodename)
}

func (grp *Group) UpdChainConfig(item *quorumpb.ChainConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdChainSendTrxRule called", grp.Item.GroupId)
	//return grp.ChainCtx.Consensus.User().UpdChainSendTrxRule(item)
	return grp.ChainCtx.Consensus.User().UpdChainConfig(item)
}

func (grp *Group) GetChainSendTrxDenyList() ([]*quorumpb.ChainConfigItem, []*quorumpb.ChainSendTrxRuleListItem, error) {
	group_log.Debugf("<%s> GetChainSendTrxDenyList called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetSendTrxAuthListByGroupId(grp.Item.GroupId, quorumpb.AuthListType_DENY_LIST, grp.ChainCtx.nodename)
}

func (grp *Group) GetChainSendTrxAllowList() ([]*quorumpb.ChainConfigItem, []*quorumpb.ChainSendTrxRuleListItem, error) {
	group_log.Debugf("<%s> GetChainSendTrxAllowList called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetSendTrxAuthListByGroupId(grp.Item.GroupId, quorumpb.AuthListType_ALLOW_LIST, grp.ChainCtx.nodename)
}

func (grp *Group) GetSendTrxAuthMode(trxType quorumpb.TrxType) (quorumpb.TrxAuthMode, error) {
	group_log.Debugf("<%s> GetSendTrxAuthMode called", grp.Item.GroupId)
	return nodectx.GetDbMgr().GetTrxAuthModeByGroupId(grp.Item.GroupId, trxType, grp.ChainCtx.nodename)
}

func (grp *Group) AskPeerId() {
	/*
		chain_log.Debugf("<%s> AskPeerId called", chain.groupId)
		var req quorumpb.AskPeerId
		req = quorumpb.AskPeerId{}

		req.GroupId = chain.groupId
		req.UserPeerId = nodectx.GetNodeCtx().Node.PeerID.Pretty()

		return chain.GetProducerTrxMgr().SendAskPeerId(&req)
	*/
}
