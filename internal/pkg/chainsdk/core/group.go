package chain

import (
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var group_log = logging.Logger("group")

const DEFAULT_EPOCH_DURATION = 1000 //ms
const INIT_FORK_TRX_ID = "00000000-0000-0000-0000-000000000000"

type Group struct {
	Item     *quorumpb.GroupItem
	ChainCtx *Chain
	GroupId  string
	Nodename string
}

func (grp *Group) NewGroup(item *quorumpb.GroupItem) (*quorumpb.Block, error) {
	group_log.Debugf("<%s> NewGroup called", item.GroupId)

	grp.Item = item
	grp.GroupId = item.GroupId
	grp.Nodename = nodectx.GetNodeCtx().Name

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.NewChain(item, grp.Nodename, false)

	ks := nodectx.GetNodeCtx().Keystore

	//create initial consensus for genesis block
	consensusInfo := &quorumpb.ConsensusInfo{
		ConsensusId:   uuid.New().String(),
		ChainVer:      0,
		InTrx:         INIT_FORK_TRX_ID,
		ForkFromBlock: 0,
	}

	//save consensus info to db
	group_log.Debugf("<%s> save consensus result", grp.Item.GroupId)
	err := nodectx.GetNodeCtx().GetChainStorage().SaveGroupConsensusInfo(item.GroupId, consensusInfo, grp.Nodename)
	if err != nil {
		group_log.Debugf("<%s> save consensus result failed", grp.Item.GroupId)
		return nil, err
	}

	//create fork trx
	forkItem := &quorumpb.ForkItem{
		GroupId:        item.GroupId,
		Consensus:      consensusInfo,
		StartFromBlock: 0,
		StartFromEpoch: 0,
		EpochDuration:  DEFAULT_EPOCH_DURATION,
		Producers:      []string{item.OwnerPubKey}, //owner is the first producer
		Memo:           "genesis fork",
	}

	forkTrx, err := grp.ChainCtx.GetTrxFactory().GetForkTrx("", forkItem)
	if err != nil {
		return nil, err
	}

	//create and save genesis block
	group_log.Debugf("<%s> create and save genesis block", grp.Item.GroupId)
	genesisBlock, err := rumchaindata.CreateGenesisBlockByEthKey(item.GroupId, consensusInfo, forkTrx, item.OwnerPubKey, ks, "")
	err = nodectx.GetNodeCtx().GetChainStorage().AddGensisBlock(genesisBlock, false, grp.Nodename)
	if err != nil {
		return nil, err
	}

	//add group owner as the first group producer
	group_log.Debugf("<%s> add owner as the first producer", grp.Item.GroupId)
	pItem := &quorumpb.ProducerItem{}
	pItem.GroupId = item.GroupId
	pItem.ProducerPubkey = item.OwnerPubKey
	pItem.ProofTrxId = ""
	pItem.BlkCnt = 0
	pItem.Memo = "Owner Registated as the first group producer"
	err = nodectx.GetNodeCtx().GetChainStorage().AddProducer(pItem, grp.Nodename)
	if err != nil {
		return nil, err
	}

	//load and update group producers
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

	//save groupItem to db
	err = nodectx.GetNodeCtx().GetChainStorage().AddGroup(grp.Item)
	if err != nil {
		return nil, err
	}

	group_log.Debugf("Group <%s> created", grp.Item.GroupId)
	return genesisBlock, nil
}

func (grp *Group) JoinGroup(item *quorumpb.GroupItem) error {
	group_log.Debugf("<%s> JoinGoup called", item.GroupId)

	grp.Item = item
	grp.GroupId = item.GroupId
	grp.Nodename = nodectx.GetNodeCtx().Name

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.NewChain(item, grp.Nodename, false)

	//get consensusInfo from genesis block
	//check there is only 1 trx(FORK) in genesis block
	if len(item.GenesisBlock.Trxs) != 1 {
		return errors.New("genesis block should have only 1 trx")
	}

	//get the trx
	forkTrx := item.GenesisBlock.Trxs[0]
	verified, err := rumchaindata.VerifyTrx(forkTrx)
	if err != nil {
		return err
	}
	if !verified {
		return errors.New("verify fork trx failed")
	}

	forkItem := &quorumpb.ForkItem{}
	err = proto.Unmarshal(forkTrx.Data, forkItem)
	if err != nil {
		return err
	}

	//save consensus info to db
	group_log.Debugf("<%s> save consensus info", grp.Item.GroupId)
	err = nodectx.GetNodeCtx().GetChainStorage().SaveGroupConsensusInfo(item.GroupId, forkItem.Consensus, grp.Nodename)
	if err != nil {
		group_log.Debugf("<%s> save consensus info failed", grp.Item.GroupId)
		return err
	}

	//save genesis block
	group_log.Debugf("<%s> save genesis block", grp.Item.GroupId)
	err = nodectx.GetNodeCtx().GetChainStorage().AddGensisBlock(item.GenesisBlock, false, grp.Nodename)
	if err != nil {
		return err
	}

	//add group owner as the first group producer
	group_log.Debugf("<%s> add owner as the first producer", grp.Item.GroupId)
	pItem := &quorumpb.ProducerItem{}
	pItem.GroupId = item.GroupId
	pItem.ProducerPubkey = item.OwnerPubKey
	pItem.ProofTrxId = ""
	pItem.BlkCnt = 0
	pItem.Memo = "Owner Registated as the first group producer"
	err = nodectx.GetNodeCtx().GetChainStorage().AddProducer(pItem, grp.Nodename)
	if err != nil {
		return err
	}

	//load and update group producers
	grp.ChainCtx.updateProducerPool()

	//create and register ConnMgr for chainctx
	conn.GetConn().RegisterChainCtx(item.GroupId,
		item.OwnerPubKey,
		item.UserSignPubkey,
		grp.ChainCtx)

	//create group consensus
	grp.ChainCtx.CreateConsensus()

	//save groupItem to db
	err = nodectx.GetNodeCtx().GetChainStorage().AddGroup(grp.Item)
	if err != nil {
		return err
	}

	group_log.Debugf("Join Group <%s> done", grp.Item.GroupId)
	return nil
}

func (grp *Group) LoadGroup(item *quorumpb.GroupItem) {
	group_log.Debugf("<%s> LoadGroup called", item.GroupId)
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

	group_log.Infof("Group <%s> loaded", grp.Item.GroupId)
}

// teardown group
func (grp *Group) Teardown() {
	group_log.Debugf("<%s> Teardown called", grp.Item.GroupId)

	//unregisted chainctx with conn
	conn.GetConn().UnregisterChainCtx(grp.Item.GroupId)

	group_log.Infof("Group <%s> teardown peacefully", grp.Item.GroupId)
}

func (grp *Group) LeaveGrp() error {
	group_log.Debugf("<%s> LeaveGrp called", grp.Item.GroupId)

	//unregisted chainctx with conn
	if err := conn.GetConn().UnregisterChainCtx(grp.Item.GroupId); err != nil {
		return err
	}

	//remove group from local db
	return nodectx.GetNodeCtx().GetChainStorage().RmGroup(grp.Item.GroupId)
}

func (grp *Group) IsProducer() bool {
	return grp.ChainCtx.IsProducer()
}

func (grp *Group) IsOwner() bool {
	return grp.ChainCtx.IsOwner()
}

func (grp *Group) ClearGroupData() error {
	group_log.Debugf("<%s> ClearGroupData called", grp.Item.GroupId)

	//remove group data from local db
	return nodectx.GetNodeCtx().GetChainStorage().RemoveGroupData(grp.Item.GroupId, grp.Nodename)
}

func (grp *Group) GetCurrentEpoch() uint64 {
	return grp.ChainCtx.GetCurrEpoch()
}

func (grp *Group) GetLatestUpdate() int64 {
	return grp.ChainCtx.GetLastUpdate()
}

func (grp *Group) GetCurrentBlockId() uint64 {
	return grp.ChainCtx.GetCurrBlockId()
}

func (grp *Group) GetNodeName() string {
	return grp.Nodename
}

func (grp *Group) GetRexSyncerStatus() string {
	return grp.ChainCtx.GetRexSyncerStatus()
}

func (grp *Group) GetBlock(blockId uint64) (blk *quorumpb.Block, isOnChain bool, err error) {
	group_log.Debugf("<%s> GetBlock called, blockId: <%d>", grp.Item.GroupId, blockId)
	block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(grp.Item.GroupId, blockId, false, grp.Nodename)
	if err == nil {
		return block, true, nil
	}
	block, err = nodectx.GetNodeCtx().GetChainStorage().GetBlock(grp.Item.GroupId, blockId, true, grp.Nodename)
	if err == nil {
		return block, false, nil
	}

	return nil, false, fmt.Errorf("GetBlock failed, block <%d> not exist", blockId)
}

func (grp *Group) GetTrx(trxId string) (tx *quorumpb.Trx, isOnChain bool, err error) {
	group_log.Debugf("<%s> GetTrx called trxId: <%s>", grp.Item.GroupId, trxId)
	trx, err := nodectx.GetNodeCtx().GetChainStorage().GetTrx(grp.Item.GroupId, trxId, def.Chain, grp.Nodename)
	if err == nil {
		return trx, true, nil
	}

	trx, err = nodectx.GetNodeCtx().GetChainStorage().GetTrx(grp.Item.GroupId, trxId, def.Cache, grp.Nodename)
	if err == nil {
		return trx, false, nil
	}

	return nil, false, fmt.Errorf("GetTrx failed, trx <%s> not exist", trxId)
}

func (grp *Group) GetProducers() ([]*quorumpb.ProducerItem, error) {
	group_log.Debugf("<%s> GetProducers called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetProducers(grp.Item.GroupId, grp.Nodename)
}

func (grp *Group) GetAppConfigKeyList() (keyName []string, itemType []string, err error) {
	group_log.Debugf("<%s> GetAppConfigKeyList called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigKey(grp.Item.GroupId, grp.Nodename)
}

func (grp *Group) GetAppConfigItem(keyName string) (*quorumpb.AppConfigItem, error) {
	group_log.Debugf("<%s> GetAppConfigItem called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigItem(keyName, grp.Item.GroupId, grp.Nodename)
}

func (grp *Group) GetCurrentTrxProposeInterval() (uint64, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GetProducerConsensusConfInterval(grp.Item.GroupId, grp.Nodename)
}

// send POST trx
func (grp *Group) PostToGroup(content []byte) (string, error) {
	group_log.Debugf("<%s> PostToGroup called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetPostAnyTrx("", content)
	if err != nil {
		return "", err
	}

	return grp.sendTrx(trx)
}

func (grp *Group) GetInitForkTrx(trxId string, item *quorumpb.ForkItem) (*quorumpb.Trx, error) {
	return grp.ChainCtx.GetTrxFactory().GetForkTrx("", item)
}

func (grp *Group) UpdGroupUser(item *quorumpb.UpdGroupUserItem) (string, error) {
	group_log.Debugf("<%s> UpdUser called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdGroupUserTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx)
}

func (grp *Group) UpdChainConfig(item *quorumpb.ChainConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdChainSendTrxRule called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetChainConfigTrx("", item)
	if err != nil {
		return "", err
	}
	return grp.sendTrx(trx)
}

// send update appconfig trx
func (grp *Group) UpdAppConfig(item *quorumpb.AppConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdAppConfig called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdAppConfigTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx)
}

// send raw trx, for light node API
func (grp *Group) SendRawTrx(trx *quorumpb.Trx) (string, error) {
	return grp.sendTrx(trx)
}

func (grp *Group) sendTrx(trx *quorumpb.Trx) (string, error) {
	connMgr, err := conn.GetConn().GetConnMgr(grp.Item.GroupId)
	if err != nil {
		return "", err
	}
	err = connMgr.SendUserTrxPubsub(trx)
	if err != nil {
		return "", err
	}

	return trx.TrxId, nil
}

func (grp *Group) StartSync() error {
	group_log.Debugf("<%s> StartSync called", grp.Item.GroupId)
	return grp.ChainCtx.StartSync()
}

func (grp *Group) StopSync() error {
	group_log.Debugf("<%s> StopSync called", grp.Item.GroupId)
	grp.ChainCtx.StopSync()
	return nil
}
