package chain

import (
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var grouprumlite_log = logging.Logger("grouprumlite")

type GroupRumLite struct {
	groupItem *quorumpb.GroupItemRumLite
	ChainCtx  *ChainRumLite
	GroupId   string
	Nodename  string
}

func (grp *GroupRumLite) JoinGroup(groupItem *quorumpb.GroupItemRumLite) error {
	grouprumlite_log.Debugf("<%s> JoinGoup called", groupItem.GroupId)

	grp.groupItem = groupItem
	grp.GroupId = groupItem.GroupId
	grp.Nodename = nodectx.GetNodeCtx().Name

	grouprumlite_log.Debugf("<%s> create and initial chainCtx", grp.GroupId)
	//create and initial chain
	grp.ChainCtx = &ChainRumLite{}
	grp.ChainCtx.NewChainRumLite(groupItem, grp.Nodename)

	grouprumlite_log.Debugf("<%s> create group consensus", grp.GroupId)
	//create group consensus
	grp.ChainCtx.CreateConsensus()

	//save genesis block
	grouprumlite_log.Debugf("<%s> save genesis block", grp.GroupId)
	err := nodectx.GetNodeCtx().GetChainStorage().AddGensisBlockRumLite(groupItem.GenesisBlock, grp.Nodename)
	if err != nil {
		return err
	}

	grouprumlite_log.Debugf("<%s> create and register ConnMgr for chainctx", grp.groupItem.GroupId)
	//create and register ConnMgr for chainctx
	conn.GetConn().RegisterChainCtx(groupItem.GroupId,
		groupItem.OwnerPubKey,
		groupItem.TrxSignPubkey,
		grp.ChainCtx)

	grouprumlite_log.Debugf("<%s> Save GroupInfo", grp.GroupId)
	//save groupItem to db
	//err = nodectx.GetNodeCtx().GetChainStorage().AddGroup(grp.Item)
	//if err != nil {
	//	return err
	//}

	grouprumlite_log.Debugf("Join Group <%s> done", grp.GroupId)
	return nil
}

func (grp *GroupRumLite) Open(groupId string) error {
	grouprumlite_log.Debugf("<%s> Open called", groupId)
	/*
		//save groupItem
		grp.Item = item
		grp.GroupId = item.GroupId
		grp.Nodename = nodectx.GetNodeCtx().Name

		//create and initial chain
		grp.ChainCtx = &Chain{}
		grp.ChainCtx.NewChain(item, grp.Nodename, true)

		opk, _ := localcrypto.Libp2pPubkeyToEthBase64(item.OwnerPubKey)
		if opk != "" {
			item.OwnerPubKey = opk
		}

		upk, _ := localcrypto.Libp2pPubkeyToEthBase64(item.UserSignPubkey)
		if upk != "" {
			item.UserSignPubkey = upk
		}

		//reload all announced user(if private)
		if grp.groupItem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			grouprumlite_log.Debugf("<%s> Private group load announced user key", grp.GroupId)
			grp.ChainCtx.updUserList()
		}

		//reload producers
		grp.ChainCtx.updateProducerPool()

		//create and register ConnMgr for chainctx
		conn.GetConn().RegisterChainCtx(item.GroupId,
			item.OwnerPubKey,
			item.UserSignPubkey,
			grp.ChainCtx)

		//commented by cuicat
		//update producer list for ConnMgr just created
		//grp.ChainCtx.UpdConnMgrProducer()

		//create group consensus
		grp.ChainCtx.CreateConsensus()

		grouprumlite_log.Infof("Group <%s> loaded", grp.groupItem.GroupId)
	*/
}

// teardown group
func (grp *GroupRumLite) Close() {
	grouprumlite_log.Debugf("<%s> Close called", grp.groupItem.GroupId)

	//unregisted chainctx with conn
	conn.GetConn().UnregisterChainCtx(grp.groupItem.GroupId)

	grouprumlite_log.Infof("Group <%s> teardown peacefully", grp.groupItem.GroupId)
}

func (grp *GroupRumLite) Delete() error {
	grouprumlite_log.Debugf("<%s> LeaveGrp called", grp.groupItem.GroupId)

	//unregisted chainctx with conn
	if err := conn.GetConn().UnregisterChainCtx(grp.groupItem.GroupId); err != nil {
		return err
	}

	//remove group from local db
	return nodectx.GetNodeCtx().GetChainStorage().RmGroup(grp.groupItem.GroupId)
}

func (grp *GroupRumLite) IsProducer() bool {
	return grp.ChainCtx.IsProducer()
}

func (grp *GroupRumLite) IsOwner() bool {
	return grp.ChainCtx.IsOwner()
}

func (grp *GroupRumLite) ClearGroupData() error {
	grouprumlite_log.Debugf("<%s> ClearGroupData called", grp.groupItem.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().RemoveGroupData(grp.groupItem.GroupId, grp.Nodename)
}

func (grp *GroupRumLite) GetCurrentEpoch() uint64 {
	return grp.ChainCtx.GetCurrEpoch()
}

func (grp *GroupRumLite) GetLatestUpdate() int64 {
	return grp.ChainCtx.GetLastUpdate()
}

func (grp *GroupRumLite) GetCurrentBlockId() uint64 {
	return grp.ChainCtx.GetCurrBlockId()
}

func (grp *GroupRumLite) GetNodeName() string {
	return grp.Nodename
}

func (grp *GroupRumLite) GetRexSyncerStatus() string {
	return grp.ChainCtx.GetRexSyncerStatus()
}

func (grp *GroupRumLite) GetBlock(blockId uint64) (blk *quorumpb.Block, isOnChain bool, err error) {
	grouprumlite_log.Debugf("<%s> GetBlock called, blockId: <%d>", grp.groupItem.GroupId, blockId)
	block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(grp.groupItem.GroupId, blockId, false, grp.Nodename)
	if err == nil {
		return block, true, nil
	}
	block, err = nodectx.GetNodeCtx().GetChainStorage().GetBlock(grp.groupItem.GroupId, blockId, true, grp.Nodename)
	if err == nil {
		return block, false, nil
	}

	return nil, false, fmt.Errorf("GetBlock failed, block <%d> not exist", blockId)
}

func (grp *GroupRumLite) GetTrx(trxId string) (tx *quorumpb.Trx, isOnChain bool, err error) {
	grouprumlite_log.Debugf("<%s> GetTrx called trxId: <%s>", grp.groupItem.GroupId, trxId)
	trx, err := nodectx.GetNodeCtx().GetChainStorage().GetTrx(grp.groupItem.GroupId, trxId, def.Chain, grp.Nodename)
	if err == nil {
		return trx, true, nil
	}

	trx, err = nodectx.GetNodeCtx().GetChainStorage().GetTrx(grp.groupItem.GroupId, trxId, def.Cache, grp.Nodename)
	if err == nil {
		return trx, false, nil
	}

	return nil, false, fmt.Errorf("GetTrx failed, trx <%s> not exist", trxId)
}

func (grp *GroupRumLite) GetProducers() ([]*quorumpb.ProducerItem, error) {
	grouprumlite_log.Debugf("<%s> GetProducers called", grp.groupItem.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetProducers(grp.groupItem.GroupId, grp.Nodename)
}

func (grp *GroupRumLite) GetAppConfigKeyList() (keyName []string, itemType []string, err error) {
	grouprumlite_log.Debugf("<%s> GetAppConfigKeyList called", grp.groupItem.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigKey(grp.groupItem.GroupId, grp.Nodename)
}

func (grp *GroupRumLite) GetAppConfigItem(keyName string) (*quorumpb.AppConfigItem, error) {
	grouprumlite_log.Debugf("<%s> GetAppConfigItem called", grp.groupItem.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigItem(keyName, grp.groupItem.GroupId, grp.Nodename)
}

func (grp *GroupRumLite) GetCurrentChainConsensus() (*quorumpb.ConsensusInfoRumLite, error) {
	return nil, nil
}

// send POST trx
func (grp *GroupRumLite) PostToGroup(content []byte) (string, error) {
	grouprumlite_log.Debugf("<%s> PostToGroup called", grp.groupItem.GroupId)

	trx, err := grp.ChainCtx.GetTrxFactory().GetPostAnyTrx(content)
	if err != nil {
		return "", err
	}

	return grp.sendTrx(trx)
}

func (grp *GroupRumLite) UpdGroupSyncer(item *quorumpb.UpdSyncerItem) (string, error) {
	grouprumlite_log.Debugf("<%s> UpdUser called", grp.groupItem.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdSyncerTrx(item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx)
}

func (grp *GroupRumLite) UpdChainConfig(item *quorumpb.ChainConfigItem) (string, error) {
	grouprumlite_log.Debugf("<%s> UpdChainSendTrxRule called", grp.groupItem.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetChainConfigTrx(item)
	if err != nil {
		return "", err
	}
	return grp.sendTrx(trx)
}

// send update appconfig trx
func (grp *GroupRumLite) UpdAppConfig(item *quorumpb.AppConfigItem) (string, error) {
	grouprumlite_log.Debugf("<%s> UpdAppConfig called", grp.groupItem.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdAppConfigTrx(item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx)
}

// send raw trx, for light node API
func (grp *GroupRumLite) SendRawTrx(trx *quorumpb.Trx) (string, error) {
	return grp.sendTrx(trx)
}

func (grp *GroupRumLite) sendTrx(trx *quorumpb.Trx) (string, error) {
	connMgr, err := conn.GetConn().GetConnMgr(grp.groupItem.GroupId)
	if err != nil {
		return "", err
	}
	err = connMgr.SendTrxPubsub(trx)
	if err != nil {
		return "", err
	}

	return trx.TrxId, nil
}

func (grp *GroupRumLite) StartSync() error {
	grouprumlite_log.Debugf("<%s> StartSync called", grp.groupItem.GroupId)
	return grp.ChainCtx.StartSync()
}

func (grp *GroupRumLite) StopSync() {
	grouprumlite_log.Debugf("<%s> StopSync called", grp.groupItem.GroupId)
	grp.ChainCtx.StopSync()
}
