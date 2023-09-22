package chain

import (
	"fmt"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

var group_log = logging.Logger("group")

type Group struct {
	ParentGroupId string
	GroupId       string
	Nodename      string

	GroupItem *quorumpb.GroupItem
	ChainCtx  *Chain
}

func (grp *Group) JoinGroupBySeed(parentGroupId, ownerKeyname, posterKeyname, producerKeyname, syncerKeyname string, seed *quorumpb.GroupSeed) error {
	group_log.Debugf("<%s> JoinGoupBySeed called", seed.GroupId)

	groupItem := &quorumpb.GroupItem{
		GroupId:       seed.GroupId,
		GroupName:     seed.GroupName,
		OwnerPubKey:   seed.OwnerPubkey,
		LastUpdate:    time.Now().UnixNano(),
		GenesisBlock:  seed.GenesisBlock,
		AuthType:      seed.AuthType,
		ConsenseType:  seed.GenesisBlock.Consensus.Type,
		CipherKey:     seed.CipherKey,
		AppId:         seed.AppId,
		AppName:       seed.AppName,
		MyOwner:       nil,
		MyPoster:      nil,
		MyProducer:    nil,
		MySyncer:      nil,
		GroupServices: nil,
	}

	grp.ParentGroupId = parentGroupId
	grp.GroupItem = groupItem
	grp.GroupId = groupItem.GroupId
	grp.Nodename = nodectx.GetNodeCtx().Name

	//save consensus info to db
	group_log.Debugf("<%s> save consensus info", grp.GroupId)
	err := nodectx.GetNodeCtx().GetChainStorage().SaveGroupConsensus(seed.GroupId, seed.GenesisBlock.Consensus, grp.Nodename)
	if err != nil {
		group_log.Debugf("<%s> save consensus info failed", grp.GroupId)
		return err
	}

	//save genesis block
	group_log.Debugf("<%s> save genesis block", grp.GroupId)
	err = nodectx.GetNodeCtx().GetChainStorage().AddGensisBlock(seed.GenesisBlock, false, grp.Nodename)
	if err != nil {
		return err
	}

	//add neo producer as the first group producer
	group_log.Debugf("<%s> add neo producer", grp.GroupId)

	pItem := &quorumpb.Producer{}
	pItem.GroupId = seed.GroupId
	pItem.ProducerPubkey = seed.GenesisBlock.ProducerPubkey
	pItem.Memo = "add group neo producer"
	err = nodectx.GetNodeCtx().GetChainStorage().AddProducer(pItem, grp.Nodename)
	if err != nil {
		return err
	}

	ks := localcrypto.GetKeystore()
	//create myPoster
	if posterKeyname != "" {
		group_log.Debugf("<%s> create myPoster", grp.GroupId)
		myPoster := &quorumpb.Poster{}
		myPoster.GroupId = seed.GroupId
		myPoster.PosterKeyname = posterKeyname
		//find pubkey by keyname
		pubkey, err := ks.GetEncodedPubkey(posterKeyname, localcrypto.Sign)
		if err != nil {
			return err
		}
		myPoster.PosterPubkey = pubkey
		myPoster.Memo = "add my group poster"
		groupItem.MyPoster = myPoster
	}

	//create myProducer
	if producerKeyname != "" {
		group_log.Debugf("<%s> create myProducer", grp.GroupId)
		myProducer := &quorumpb.Producer{}
		myProducer.GroupId = seed.GroupId
		myProducer.ProducerKeyanme = producerKeyname
		//find pubkey by keyname
		pubkey, err := ks.GetEncodedPubkey(producerKeyname, localcrypto.Sign)
		if err != nil {
			return err
		}
		myProducer.ProducerPubkey = pubkey
		myProducer.Memo = "add my group producer"
		groupItem.MyProducer = myProducer
	}

	//create mySyncer
	if syncerKeyname != "" {
		group_log.Debugf("<%s> create mySyncer", grp.GroupId)
		mySyncer := &quorumpb.Syncer{}
		mySyncer.GroupId = seed.GroupId
		mySyncer.SyncerKeyname = syncerKeyname
		//find pubkey by keyname
		pubkey, err := ks.GetEncodedPubkey(syncerKeyname, localcrypto.Sign)
		if err != nil {
			return err
		}
		mySyncer.SyncerPubkey = pubkey
		mySyncer.Memo = "add my group syncer"
		groupItem.MySyncer = mySyncer
	}

	//create myOwner (if has owner keyname)
	if ownerKeyname != "" {
		group_log.Debugf("<%s> create myOwner", grp.GroupId)
		myOwner := &quorumpb.Owner{}
		myOwner.GroupId = seed.GroupId
		myOwner.OwnerKeyname = ownerKeyname
		//find pubkey by keyname
		pubkey, err := ks.GetEncodedPubkey(ownerKeyname, localcrypto.Sign)
		if err != nil {
			return err
		}

		//check if my owner key is as same as group owner key
		if pubkey != seed.OwnerPubkey {
			return fmt.Errorf("do you really has the group owner keypair? check your local keystore!")
		}
		myOwner.OwnerPubkey = pubkey
		myOwner.Memo = "add my group owner"
		groupItem.MyOwner = myOwner
	}

	//parse group service items
	groupServices := &quorumpb.GroupServices{}
	group_log.Debugf("<%s> save group service items", grp.GroupItem.GroupId)
	if seed.Services != nil {
		for _, service := range seed.Services {
			if service.TaskType == quorumpb.GroupTaskType_PRODUCE {
				produceService := &quorumpb.ProduceServiceItem{}
				err = proto.Unmarshal(service.Data, produceService)
				if err != nil {
					return err
				}
				groupServices.ProduceService = produceService
			} else if service.TaskType == quorumpb.GroupTaskType_SYNC {
				syncService := &quorumpb.SyncServiceItem{}
				err = proto.Unmarshal(service.Data, syncService)
				if err != nil {
					return err
				}
				groupServices.SyncService = syncService
			} else if service.TaskType == quorumpb.GroupTaskType_PUBLISH {
				publishService := &quorumpb.PublishServiceItem{}
				err = proto.Unmarshal(service.Data, publishService)
				if err != nil {
					return err
				}
				groupServices.PublishService = publishService
			} else if service.TaskType == quorumpb.GroupTaskType_CTN {
				ctnService := &quorumpb.CtnServiceItem{}
				err = proto.Unmarshal(service.Data, ctnService)
				if err != nil {
					return err
				}
				groupServices.CtnService = ctnService
			}
		}
	}

	groupItem.GroupServices = groupServices

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.NewChainWithSeed(seed, groupItem, grp, grp.Nodename)

	//load and update group producers
	grp.ChainCtx.UpdateProducerPool()

	//create and register ConnMgr for chainctx
	conn.GetConn().RegisterChainCtx(groupItem.GroupId, grp.ChainCtx)

	//create group consensus
	grp.ChainCtx.CreateConsensus()

	//save groupItem to db
	err = nodectx.GetNodeCtx().GetChainStorage().AddGroup(grp.ParentGroupId, grp.GroupItem)
	if err != nil {
		return err
	}

	//save group to groupMgr
	GetGroupMgr().AddLocalGroup(grp)

	group_log.Debugf("Join Group <%s> done", grp.GroupId)
	return nil
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

func (grp *Group) LoadGroup(parentGroupId string, item *quorumpb.GroupItem) error {
	group_log.Debugf("<%s> LoadGroup called", item.GroupId)
	//save groupItem
	grp.ParentGroupId = parentGroupId
	grp.GroupItem = item
	grp.GroupId = item.GroupId
	grp.Nodename = nodectx.GetNodeCtx().Name

	//create and initial chain
	grp.ChainCtx = &Chain{}
	err := grp.ChainCtx.LoadChain(item, grp.Nodename)
	if err != nil {
		return err
	}

	//reload producers
	grp.ChainCtx.UpdateProducerPool()

	//create and register ConnMgr for chainctx
	conn.GetConn().RegisterChainCtx(item.GroupId, grp.ChainCtx)

	//create group consensus
	grp.ChainCtx.CreateConsensus()
	group_log.Infof("Group <%s> loaded", grp.GroupId)

	//save group to groupMgr
	GetGroupMgr().AddLocalGroup(grp)

	return nil
}

// teardown group
func (grp *Group) Teardown() error {
	group_log.Debugf("<%s> Teardown called", grp.GroupId)

	//unregisted chainctx with conn
	if err := conn.GetConn().UnregisterChainCtx(grp.GroupId); err != nil {
		group_log.Debugf("<%s> UnregisterChainCtx failed", grp.GroupId)
		return err
	}

	//cancel ctx
	grp.ChainCtx.CtxCancelFunc()
	group_log.Infof("Group <%s> teardown peacefully", grp.GroupId)

	return nil
}

func (grp *Group) StartAllMyServics() error {
	group_log.Debugf("<%s> StartAllServics called", grp.GroupItem.GroupId)
	return grp.ChainCtx.StartSync()
}

func (grp *Group) StopAllMyServices() error {
	group_log.Debugf("<%s> StopAllServices called", grp.GroupItem.GroupId)
	grp.ChainCtx.StopSync()
	return nil
}

func (grp *Group) ClearGroupData() error {
	group_log.Debugf("<%s> ClearGroupData called", grp.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().RemoveGroupData(grp.GroupId, grp.Nodename)
}

func (grp *Group) GetGroupId() string {
	return grp.GroupId
}

func (grp *Group) GetNodeName() string {
	return grp.Nodename
}

func (grp *Group) GetOwnerPubkey() string {
	return grp.GroupItem.OwnerPubKey
}

func (grp *Group) GetGroupName() string {
	return grp.GroupItem.GroupName
}

func (grp *Group) GetConsensusType() string {
	return grp.GroupItem.ConsenseType.String()
}

func (grp *Group) GetAuthType() string {
	return grp.GroupItem.AuthType.String()
}

func (grp *Group) GetAppId() string {
	return grp.GroupItem.AppId
}

func (grp *Group) GetAppName() string {
	return grp.GroupItem.AppName
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

func (grp *Group) GetRexSyncerStatus() string {
	return grp.ChainCtx.GetRexSyncerStatus()
}

func (grp *Group) GetLastUpdated() int64 {
	return grp.ChainCtx.GetLastUpdate()
}

func (grp *Group) GetBlock(blockId uint64) (blk *quorumpb.Block, isOnChain bool, err error) {
	group_log.Debugf("<%s> GetBlock called, blockId: <%d>", grp.GroupId, blockId)
	block, err := nodectx.GetNodeCtx().GetChainStorage().GetBlock(grp.GroupId, blockId, false, grp.Nodename)
	if err == nil {
		return block, true, nil
	}
	block, err = nodectx.GetNodeCtx().GetChainStorage().GetBlock(grp.GroupId, blockId, true, grp.Nodename)
	if err == nil {
		return block, false, nil
	}

	return nil, false, fmt.Errorf("GetBlock failed, block <%d> not exist", blockId)
}

func (grp *Group) GetTrx(trxId string) (tx *quorumpb.Trx, isOnChain bool, err error) {
	group_log.Debugf("<%s> GetTrx called trxId: <%s>", grp.GroupId, trxId)
	trx, err := nodectx.GetNodeCtx().GetChainStorage().GetTrx(grp.GroupId, trxId, def.Chain, grp.Nodename)
	if err == nil {
		return trx, true, nil
	}
	trx, err = nodectx.GetNodeCtx().GetChainStorage().GetTrx(grp.GroupId, trxId, def.Cache, grp.Nodename)
	if err == nil {
		return trx, false, nil
	}

	return nil, false, fmt.Errorf("GetTrx failed, trx <%s> not exist", trxId)
}

func (grp *Group) GetCipherKey() string {
	return grp.GroupItem.CipherKey
}

func (grp *Group) GetProducers() ([]*quorumpb.Producer, error) {
	group_log.Debugf("<%s> GetProducers called", grp.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetProducers(grp.GroupId, grp.Nodename)
}

func (grp *Group) GetAppConfigKeyList() (keyName []string, itemType []string, err error) {
	group_log.Debugf("<%s> GetAppConfigKeyList called", grp.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigKey(grp.GroupId, grp.Nodename)
}

func (grp *Group) GetAppConfigItem(keyName string) (*quorumpb.AppConfigItem, error) {
	group_log.Debugf("<%s> GetAppConfigItem called", grp.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigItem(keyName, grp.GroupId, grp.Nodename)
}

func (grp *Group) GetCurrentConsensus() (*quorumpb.Consensus, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GetGroupConsensus(grp.GroupId, grp.Nodename)
}

func (grp *Group) PostToGroup(content []byte) (string, error) {
	group_log.Debugf("<%s> PostToGroup called", grp.GroupId)

	if grp.GroupItem.MyPoster == nil {
		return "", fmt.Errorf("myPoster not exist")
	}

	trx, err := grp.ChainCtx.GetTrxFactory().GetPostAnyTrx(grp.GroupItem.MyPoster.PosterKeyname, grp.GroupItem.MyPoster.PosterPubkey, content)
	if err != nil {
		return "", err
	}

	return grp.sendTrx(trx)
}

func (grp *Group) UpdGroupSyncer(item *quorumpb.UpdGroupSyncerItem) (string, error) {
	group_log.Debugf("<%s> UpdGroupSyncer called", grp.GroupItem.GroupId)

	if grp.GroupItem.MyOwner == nil {
		return "", fmt.Errorf("myOwner not exist")
	}

	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdGroupSyncerTrx(grp.GroupItem.MyOwner.OwnerPubkey, grp.GroupItem.MyOwner.OwnerKeyname, item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx)
}

func (grp *Group) UpdGroupPoster(item *quorumpb.UpdGroupPosterItem) (string, error) {
	group_log.Debugf("<%s> UpdGroupPoster called", grp.GroupItem.GroupId)
	if grp.GroupItem.MyOwner == nil {
		return "", fmt.Errorf("myOwner not exist")
	}
	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdGroupPosterTrx(grp.GroupItem.MyOwner.OwnerPubkey, grp.GroupItem.MyOwner.OwnerKeyname, item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx)
}

// send update appconfig trx
func (grp *Group) UpdAppConfig(item *quorumpb.AppConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdAppConfig called", grp.GroupId)
	if grp.GroupItem.MyOwner == nil {
		return "", fmt.Errorf("myOwner not exist")
	}

	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdAppConfigTrx(grp.GroupItem.MyOwner.OwnerKeyname, grp.GroupItem.MyOwner.OwnerKeyname, item)
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
	connMgr, err := conn.GetConn().GetConnMgr(grp.GroupItem.GroupId)
	if err != nil {
		return "", err
	}
	err = connMgr.SendUserTrxPubsub(trx)
	if err != nil {
		return "", err
	}

	return trx.TrxId, nil
}

func (grp *Group) SendTrxBySeed(trx *quorumpb.Trx, cellarSeed *quorumpb.GroupSeed) (string, error) {
	connMgr, err := conn.GetConn().GetConnMgr(grp.GroupItem.GroupId)
	if err != nil {
		return "", err
	}
	err = connMgr.SendTrxBySeed(cellarSeed, trx)
	if err != nil {
		return "", err
	}
	return trx.TrxId, nil
}

func (grp *Group) ReqCellarServices(cellarSeedByts []byte, serviceType quorumpb.GroupTaskType, proof []byte, memo string) (string, error) {
	group_log.Debugf("<%s> ReqCellarServices called", grp.GroupId)
	/*
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
		myseed, err := nodectx.GetNodeCtx().GetChainStorage().GetGroupSeed(grp.GroupItem.GroupId)
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
	*/
	return "", nil
}
