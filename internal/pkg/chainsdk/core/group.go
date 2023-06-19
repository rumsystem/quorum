package chain

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"

	guuid "github.com/google/uuid"
)

var group_log = logging.Logger("group")

type Group struct {
	Item     *quorumpb.GroupItem
	ChainCtx *Chain
	GroupId  string
	Nodename string
}

func (grp *Group) NewGroup(item *quorumpb.GroupItem) error {
	group_log.Debugf("<%s> NewGroup called", item.GroupId)

	grp.Item = item
	grp.GroupId = item.GroupId
	grp.Nodename = nodectx.GetNodeCtx().Name

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.NewChain(item, grp.Nodename, false)

	//save group genesis block
	group_log.Debugf("<%s> save genesis block", grp.Item.GroupId)
	err := nodectx.GetNodeCtx().GetChainStorage().AddGensisBlock(item.GenesisBlock, false, grp.Nodename)
	if err != nil {
		return err
	}

	group_log.Debugf("<%s> owner announce as the first group producer", grp.Item.GroupId)
	//owner announce as the first group producer
	aContent := &quorumpb.AnnounceContent{
		Type:          quorumpb.AnnounceType_AS_PRODUCER,
		SignPubkey:    item.OwnerPubKey,
		EncryptPubkey: item.UserEncryptPubkey,
		Memo:          "owner announce as the first group producer",
	}

	aItem := &quorumpb.AnnounceItem{
		GroupId:         item.GroupId,
		Action:          quorumpb.ActionType_ADD,
		Content:         aContent,
		AnnouncerPubkey: item.OwnerPubKey,
	}

	//create hash
	byts, err := proto.Marshal(aItem)
	if err != nil {
		return err
	}
	aItem.Hash = localcrypto.Hash(byts)

	ks := nodectx.GetNodeCtx().Keystore
	signature, err := ks.EthSignByKeyName(item.GroupId, aItem.Hash)

	if err != nil {
		return err
	}

	aItem.Signature = signature

	//save aItem to db
	err = nodectx.GetNodeCtx().GetChainStorage().AddAnnounceItem(aItem, grp.Nodename)
	if err != nil {
		return err
	}

	//add consensus info for group owner
	//create consensus req
	consensusReq := &quorumpb.ChangeConsensusReq{
		ReqId:                guuid.New().String(),
		GroupId:              item.GroupId,
		Nonce:                0,
		ProducerPubkeyList:   []string{item.OwnerPubKey},
		AgreementTickLenInMs: 0,
		AgreementTickCount:   0,
		StartFromEpoch:       0,
		TrxEpochTickLenInMs:  uint64(DEFAULT_PROPOSE_TRX_INTERVAL),
		Contract:             nil,
		SenderPubkey:         item.OwnerPubKey,
	}

	//create hash for consensus req
	byts, err = proto.Marshal(consensusReq)
	if err != nil {
		return err
	}

	consensusReq.MsgHash = localcrypto.Hash(byts)

	//sign hash
	signature, err = ks.EthSignByKeyName(item.GroupId, consensusReq.MsgHash)
	if err != nil {
		return err
	}

	consensusReq.SenderSign = signature

	//creaet consensus resp
	consensusResp := &quorumpb.ChangeConsensusResp{
		RespId:       guuid.New().String(),
		GroupId:      item.GroupId,
		SenderPubkey: item.OwnerPubKey,
		Req:          consensusReq,
	}

	//create hash for consensus resp
	byts, err = proto.Marshal(consensusResp)
	if err != nil {
		return err
	}

	consensusResp.MsgHash = localcrypto.Hash(byts)
	//sign hash
	signature, err = ks.EthSignByKeyName(item.GroupId, consensusResp.MsgHash)
	if err != nil {
		return err
	}

	consensusResp.SenderSign = signature
	//create ResultBundle
	resultBundle := &quorumpb.ChangeConsensusResultBundle{
		Result:             quorumpb.ChangeConsensusResult_SUCCESS,
		Req:                consensusReq,
		Resps:              []*quorumpb.ChangeConsensusResp{consensusResp},
		ResponsedProducers: []string{item.OwnerPubKey},
	}

	group_log.Debugf("<%s> save consensus result", grp.Item.GroupId)
	//save resultBundle to db
	err = nodectx.GetNodeCtx().GetChainStorage().UpdateChangeConsensusResult(item.GroupId, resultBundle, grp.Nodename)
	if err != nil {
		group_log.Debugf("<%s> save consensus result failed", grp.Item.GroupId)
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

	//commented by cuicat
	//update producer list for ConnMgr just created
	//grp.ChainCtx.UpdConnMgrProducer()

	//create group consensus
	grp.ChainCtx.CreateConsensus()

	//save groupItem to db
	err = nodectx.GetNodeCtx().GetChainStorage().AddGroup(grp.Item)
	if err != nil {
		return err
	}

	group_log.Debugf("Group <%s> created", grp.Item.GroupId)
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

	//reload all announced user(if private)
	if grp.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		group_log.Debugf("<%s> Private group load announced user key", grp.GroupId)
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

func (grp *Group) GetAnnouncedProducer(pubkey string) (*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedProducer called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAnnouncedProducer(grp.Item.GroupId, pubkey, grp.Nodename)
}

func (grp *Group) GetAnnouncedUser(pubkey string) (*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedUser called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAnnouncedUser(grp.Item.GroupId, pubkey, grp.Nodename)
}

func (grp *Group) GetAppConfigKeyList() (keyName []string, itemType []string, err error) {
	group_log.Debugf("<%s> GetAppConfigKeyList called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigKey(grp.Item.GroupId, grp.Nodename)
}

func (grp *Group) GetAppConfigItem(keyName string) (*quorumpb.AppConfigItem, error) {
	group_log.Debugf("<%s> GetAppConfigItem called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigItem(keyName, grp.Item.GroupId, grp.Nodename)
}

func (grp *Group) GetAllChangeConsensusResultBundle() ([]*quorumpb.ChangeConsensusResultBundle, error) {
	group_log.Debugf("<%s> GetAllChangeConsensusResultBundle called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAllChangeConsensusResult(grp.Item.GroupId, grp.Nodename)
}

func (grp *Group) GetCurrentTrxProposeInterval() (uint64, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GetProducerConsensusConfInterval(grp.Item.GroupId, grp.Nodename)
}

func (grp *Group) GetLastChangeConsensusResult(isSuccess bool) (*quorumpb.ChangeConsensusResultBundle, error) {
	group_log.Debugf("<%s> GetLastSuccessChangeConsensusResult called", grp.Item.GroupId)
	results, err := nodectx.GetNodeCtx().GetChainStorage().GetAllChangeConsensusResult(grp.Item.GroupId, grp.Nodename)
	if err != nil {
		return nil, err
	}

	//if there is only 1 proof and nonce is 0, return it (added by owner when create group)
	if len(results) == 1 && results[0].Req.Nonce == 0 {
		return results[0], nil
	}

	nonce := uint64(0)
	last := &quorumpb.ChangeConsensusResultBundle{}
	for _, result := range results {
		if isSuccess && result.Result != quorumpb.ChangeConsensusResult_SUCCESS {
			continue
		}
		if result.Req.Nonce > nonce {
			last = result
			nonce = result.Req.Nonce
		}
	}
	return last, nil
}

func (grp *Group) GetChangeConsensusResultById(id string) (*quorumpb.ChangeConsensusResultBundle, error) {
	group_log.Debugf("<%s> GetChangeConsensusResultById called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetChangeConsensusResultByReqId(grp.Item.GroupId, id, grp.Nodename)
}

// send update announce trx
func (grp *Group) UpdAnnounce(item *quorumpb.AnnounceItem) (string, error) {
	group_log.Debugf("<%s> UpdAnnounce called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetAnnounceTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx)
}

// send POST trx
func (grp *Group) PostToGroup(content []byte) (string, error) {
	group_log.Debugf("<%s> PostToGroup called", grp.Item.GroupId)
	if grp.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		keys, err := grp.ChainCtx.GetUsesEncryptPubKeys()
		if err != nil {
			return "", err
		}

		trx, err := grp.ChainCtx.GetTrxFactory().GetPostAnyTrx("", content, keys)
		if err != nil {
			return "", err
		}
		return grp.sendTrx(trx)
	}

	trx, err := grp.ChainCtx.GetTrxFactory().GetPostAnyTrx("", content)
	if err != nil {
		return "", err
	}

	return grp.sendTrx(trx)
}

func (grp *Group) ReqChangeConsensus(producers []string, agrmTickLength, agrmTickCount, fromBlock uint64, fromEpoch uint64, epoch uint64) (string, uint64, error) {
	group_log.Debugf("<%s> ReqChangeConsensus called", grp.Item.GroupId)
	return grp.ChainCtx.ReqChangeConsensus(producers, agrmTickLength, agrmTickCount, fromBlock, fromEpoch, epoch)
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
