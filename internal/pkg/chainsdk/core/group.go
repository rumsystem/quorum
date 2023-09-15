package chain

import (
	"fmt"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	"github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var group_log = logging.Logger("group")

type Group struct {
	ParentGroupId string
	Item          *quorumpb.GroupItem
	BrewService   *quorumpb.BrewServiceItem
	SyncService   *quorumpb.SyncServiceItem
	ChainCtx      *Chain
	GroupId       string
	Nodename      string
}

func (grp *Group) JoinGroupBySeed(parentGroupId, userPubkey string, seed *quorumpb.GroupSeed) error {
	group_log.Debugf("<%s> JoinGoupBySeed called", seed.GroupId)

	groupItem := &quorumpb.GroupItem{
		GroupId:        seed.GroupId,
		GroupName:      seed.GroupName,
		OwnerPubKey:    seed.OwnerPubkey,
		UserSignPubkey: userPubkey,
		LastUpdate:     time.Now().UnixNano(),
		GenesisBlock:   seed.GenesisBlock,
		SyncType:       seed.SyncType,
		ConsenseType:   seed.GenesisBlock.Consensus.Type,
		CipherKey:      seed.CipherKey,
		AppId:          seed.AppId,
		AppName:        seed.AppName,
	}

	grp.ParentGroupId = parentGroupId
	grp.Item = groupItem
	grp.GroupId = groupItem.GroupId
	grp.Nodename = nodectx.GetNodeCtx().Name

	//save consensus info to db
	group_log.Debugf("<%s> save consensus info", grp.Item.GroupId)
	err := nodectx.GetNodeCtx().GetChainStorage().SaveGroupConsensus(seed.GroupId, seed.GenesisBlock.Consensus, grp.Nodename)
	if err != nil {
		group_log.Debugf("<%s> save consensus info failed", grp.Item.GroupId)
		return err
	}

	//save genesis block
	group_log.Debugf("<%s> save genesis block", grp.Item.GroupId)
	err = nodectx.GetNodeCtx().GetChainStorage().AddGensisBlock(seed.GenesisBlock, false, grp.Nodename)
	if err != nil {
		return err
	}

	//add neo producer as the first group producer
	group_log.Debugf("<%s> add neo producer", grp.Item.GroupId)
	pItem := &quorumpb.ProducerItem{}
	pItem.GroupId = seed.GroupId
	pItem.ProducerPubkey = seed.GenesisBlock.ProducerPubkey
	pItem.ProofTrxId = ""
	pItem.BlkCnt = 0
	pItem.Memo = "add group neo producer"
	err = nodectx.GetNodeCtx().GetChainStorage().AddProducer(pItem, grp.Nodename)
	if err != nil {
		return err
	}

	//set group services
	grp.BrewService = nil
	grp.SyncService = nil

	//parse group service items
	group_log.Debugf("<%s> save group service items", grp.Item.GroupId)
	if seed.Services != nil {
		for _, service := range seed.Services {
			if service.Type == quorumpb.GroupServiceType_BREW_SERVICE {
				brewService := &quorumpb.BrewServiceItem{}
				err = proto.Unmarshal(service.Service, brewService)
				if err != nil {
					return err
				}
				grp.BrewService = brewService
				err = nodectx.GetNodeCtx().GetChainStorage().SaveGroupBrewService(grp.GroupId, brewService, grp.Nodename)
				if err != nil {
					return err
				}
			} else if service.Type == quorumpb.GroupServiceType_SYNC_SERVICE {
				syncService := &quorumpb.SyncServiceItem{}
				err = proto.Unmarshal(service.Service, syncService)
				if err != nil {
					return err
				}
				grp.SyncService = syncService
				err = nodectx.GetNodeCtx().GetChainStorage().SaveGroupSyncService(grp.GroupId, syncService, grp.Nodename)
				if err != nil {
					return err
				}
			}
		}
	}

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.NewChainWithSeed(seed, groupItem, grp.Nodename)

	//load and update group producers
	grp.ChainCtx.updateProducerPool()

	//create and register ConnMgr for chainctx
	conn.GetConn().RegisterChainCtx(groupItem.GroupId,
		groupItem.OwnerPubKey,
		groupItem.UserSignPubkey,
		grp.ChainCtx)

	//create group consensus
	grp.ChainCtx.CreateConsensus()

	//save groupItem to db
	err = nodectx.GetNodeCtx().GetChainStorage().AddGroup(grp.ParentGroupId, grp.Item)
	if err != nil {
		return err
	}

	group_log.Debugf("Join Group <%s> done", grp.Item.GroupId)
	return nil
}

func (grp *Group) LoadGroup(parentGroupId string, item *quorumpb.GroupItem) {
	group_log.Debugf("<%s> LoadGroup called", item.GroupId)
	//save groupItem
	grp.ParentGroupId = parentGroupId
	grp.Item = item
	grp.GroupId = item.GroupId
	grp.Nodename = nodectx.GetNodeCtx().Name

	//set group services
	grp.BrewService = nil
	grp.SyncService = nil

	brewService, err := nodectx.GetNodeCtx().GetChainStorage().GetGroupBrewService(grp.GroupId, grp.Nodename)
	if err == nil {
		grp.BrewService = brewService
	}

	syncService, err := nodectx.GetNodeCtx().GetChainStorage().GetGroupSyncService(grp.GroupId, grp.Nodename)
	if err == nil {
		grp.SyncService = syncService
	}

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.LoadChain(item, grp.Nodename)

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

	//create group consensus
	grp.ChainCtx.CreateConsensus()
	group_log.Infof("Group <%s> loaded", grp.Item.GroupId)
}

func (grp *Group) LoadGroupById(parentGroupId, groupId string) error {
	//load groupitem by groupid
	groupItem, err := nodectx.GetNodeCtx().GetChainStorage().GetGroupItem(parentGroupId, groupId)
	if err != nil {
		return err
	}

	grp.LoadGroup(parentGroupId, groupItem)
	return nil
}

// teardown group
func (grp *Group) Teardown() error {
	group_log.Debugf("<%s> Teardown called", grp.Item.GroupId)

	//unregisted chainctx with conn
	if err := conn.GetConn().UnregisterChainCtx(grp.Item.GroupId); err != nil {
		group_log.Debugf("<%s> UnregisterChainCtx failed", grp.Item.GroupId)
		return err
	}

	//cancel ctx
	grp.ChainCtx.CtxCancelFunc()
	group_log.Infof("Group <%s> teardown peacefully", grp.Item.GroupId)

	return nil
}

/*
func (grp *Group) LeaveGrp() error {
	group_log.Debugf("<%s> LeaveGrp called", grp.Item.GroupId)

	//unregisted chainctx with conn
	if err := conn.GetConn().UnregisterChainCtx(grp.Item.GroupId); err != nil {
		return err
	}

	//cancel ctx
	grp.ChainCtx.CtxCancelFunc()

	//remove group from local db
	return nodectx.GetNodeCtx().GetChainStorage().RmGroup(grp.Item.GroupId)
}
*/

func (grp *Group) GetGroupId() string {
	return grp.Item.GroupId
}

func (grp *Group) IsProducer() bool {
	return grp.ChainCtx.IsProducer()
}

func (grp *Group) HasOwnerKey() bool {
	return grp.ChainCtx.HasOwnerKey()
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

func (grp *Group) GetCurrentConsensus() (*quorumpb.Consensus, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GetGroupConsensus(grp.Item.GroupId, grp.Nodename)
}

// send POST trx
func (grp *Group) PostToGroup(content []byte) (string, error) {
	group_log.Debugf("<%s> PostToGroup called", grp.Item.GroupId)

	signKeyName := grp.ChainCtx.GetKeynameByPubkey(grp.Item.UserSignPubkey)
	if signKeyName == "" {
		group_log.Debugf("<%s> PostToGroup failed, sign key not exist", grp.Item.GroupId)
		return "", fmt.Errorf("sign key not exist")
	}

	trx, err := grp.ChainCtx.GetTrxFactory().GetPostAnyTrx(grp.Item.UserSignPubkey, signKeyName, content)
	if err != nil {
		return "", err
	}

	return grp.sendTrx(trx)
}

func (grp *Group) UpdGroupSyncer(item *quorumpb.UpdGroupSyncerItem) (string, error) {
	group_log.Debugf("<%s> UpdGroupSyncer called", grp.Item.GroupId)

	signKeyName := grp.ChainCtx.GetKeynameByPubkey(grp.Item.OwnerPubKey)
	if signKeyName == "" {
		group_log.Debugf("<%s> UpdGroupSyncer failed, sign key not exist", grp.Item.GroupId)
		return "", fmt.Errorf("owner sign key not exist")
	}

	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdGroupSyncerTrx(grp.Item.OwnerPubKey, signKeyName, item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx)
}

func (grp *Group) ReqCellarServices(cellarSeedByts []byte, serviceType quorumpb.GroupServiceType, proof []byte, memo string) (string, error) {
	group_log.Debugf("<%s> ReqCellarServices called", grp.Item.GroupId)

	//unmarshall cellardbytes to groupseed
	cellarSeed := &quorumpb.GroupSeed{}
	err := proto.Unmarshal(cellarSeedByts, cellarSeed)
	if err != nil {
		return "", err
	}

	verified, err := data.VerifyGroupSeed(cellarSeed)
	if err != nil {
		return "", err
	}

	if !verified {
		return "", fmt.Errorf("seed not verified")
	}

	//get my group seed
	myseed, err := nodectx.GetNodeCtx().GetChainStorage().GetGroupSeed(grp.Item.GroupId)
	if err != nil {
		return "", err
	}

	//marshall myseed to bytes
	myseedByts, err := proto.Marshal(myseed)
	if err != nil {
		return "", err
	}

	req := &quorumpb.AddCellarReqItem{}
	req.Seed = myseedByts
	req.CurrentBlockId = grp.GetCurrentBlockId()
	req.Proof = proof
	req.Memo = memo

	//create addcellarreq trx
	signKeyName := grp.ChainCtx.GetKeynameByPubkey(grp.Item.OwnerPubKey)
	if signKeyName == "" {
		group_log.Debugf("<%s> AddGroupCellar failed, sign key not exist", grp.Item.GroupId)
		return "", fmt.Errorf("sign key not exist")
	}

	trx, err := grp.ChainCtx.GetTrxFactory().GetAddCellarReqTrx(grp.Item.OwnerPubKey, signKeyName, cellarSeed.CipherKey, req)
	if err != nil {
		return "", err
	}

	return grp.sendTrxToCellar(trx, cellarSeed)
}

func (grp *Group) UpdChainConfig(item *quorumpb.ChainConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdChainSendTrxRule called", grp.Item.GroupId)

	signKeyName := grp.ChainCtx.GetKeynameByPubkey(grp.Item.OwnerPubKey)
	if signKeyName == "" {
		group_log.Debugf("<%s> UpdChainConfig failed, sign key not exist", grp.Item.GroupId)
		return "", fmt.Errorf("sign key not exist")
	}

	trx, err := grp.ChainCtx.GetTrxFactory().GetChainConfigTrx(grp.Item.OwnerPubKey, signKeyName, item)
	if err != nil {
		return "", err
	}
	return grp.sendTrx(trx)
}

// send update appconfig trx
func (grp *Group) UpdAppConfig(item *quorumpb.AppConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdAppConfig called", grp.Item.GroupId)

	signKeyName := grp.ChainCtx.GetKeynameByPubkey(grp.Item.OwnerPubKey)
	if signKeyName == "" {
		group_log.Debugf("<%s> UpdAppConfig failed, sign key not exist", grp.Item.GroupId)
		return "", fmt.Errorf("sign key not exist")
	}

	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdAppConfigTrx(grp.Item.OwnerPubKey, signKeyName, item)
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

func (grp *Group) sendTrxToCellar(trx *quorumpb.Trx, cellarSeed *quorumpb.GroupSeed) (string, error) {
	connMgr, err := conn.GetConn().GetConnMgr(grp.Item.GroupId)
	if err != nil {
		return "", err
	}
	err = connMgr.SendTrxToCellar(cellarSeed, trx)
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
