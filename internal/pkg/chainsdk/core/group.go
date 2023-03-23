package chain

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

var group_log = logging.Logger("group")

type Group struct {
	//Group Item
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

	//add group owner as the first group producer
	group_log.Debugf("<%s> add owner as the first producer", grp.Item.GroupId)
	pItem := &quorumpb.ProducerItem{}
	pItem.GroupId = item.GroupId
	pItem.GroupOwnerPubkey = item.OwnerPubKey
	pItem.ProducerPubkey = item.OwnerPubKey

	var buffer bytes.Buffer
	buffer.Write([]byte(pItem.GroupId))
	buffer.Write([]byte(pItem.ProducerPubkey))
	buffer.Write([]byte(pItem.GroupOwnerPubkey))
	hash := localcrypto.Hash(buffer.Bytes())

	ks := nodectx.GetNodeCtx().Keystore
	signature, err := ks.EthSignByKeyName(item.GroupId, hash)
	if err != nil {
		return err
	}
	pItem.GroupOwnerSign = hex.EncodeToString(signature)
	pItem.Memo = "Owner Registated as the first group producer"
	pItem.TimeStamp = time.Now().UnixNano()

	err = nodectx.GetNodeCtx().GetChainStorage().AddProducer(pItem, grp.Nodename)
	if err != nil {
		return err
	}

	//load and update group producers
	grp.ChainCtx.updProducerList()

	//create and register ConnMgr for chainctx
	conn.GetConn().RegisterChainCtx(item.GroupId,
		item.OwnerPubKey,
		item.UserSignPubkey,
		grp.ChainCtx)

	//update producer list for ConnMgr just created
	grp.ChainCtx.UpdConnMgrProducer()

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
	grp.ChainCtx.updProducerList()

	//create and register ConnMgr for chainctx
	conn.GetConn().RegisterChainCtx(item.GroupId,
		item.OwnerPubKey,
		item.UserSignPubkey,
		grp.ChainCtx)

	//update producer list for ConnMgr just created
	grp.ChainCtx.UpdConnMgrProducer()

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

func (grp *Group) GetBlock(blockId uint64) (*quorumpb.Block, error) {
	group_log.Debugf("<%s> GetBlock called, blockId: <%d>", grp.Item.GroupId, blockId)
	return nodectx.GetNodeCtx().GetChainStorage().GetBlock(grp.Item.GroupId, blockId, false, grp.Nodename)
}

func (grp *Group) GetTrx(trxId string) (*quorumpb.Trx, error) {
	group_log.Debugf("<%s> GetTrx called trxId: <%s>", grp.Item.GroupId, trxId)
	return nodectx.GetNodeCtx().GetChainStorage().GetTrx(grp.Item.GroupId, trxId, def.Chain, grp.Nodename)
}

func (grp *Group) GetTrxFromCache(trxId string) (*quorumpb.Trx, error) {
	group_log.Debugf("<%s> GetTrxFromCache called trxId: <%s>", grp.Item.GroupId, trxId)
	return nodectx.GetNodeCtx().GetChainStorage().GetTrx(grp.Item.GroupId, trxId, def.Cache, grp.Nodename)
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

func (grp *Group) UpdProducer(item *quorumpb.BFTProducerBundleItem) (string, error) {
	group_log.Debugf("<%s> UpdProducer called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetRegProducerBundleTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx)
}

func (grp *Group) UpdUser(item *quorumpb.UserItem) (string, error) {
	group_log.Debugf("<%s> UpdUser called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetRegUserTrx("", item)
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

func (grp *Group) StartSync(restart bool) error {
	group_log.Debugf("<%s> StartSync called", grp.Item.GroupId)
	return grp.ChainCtx.StartSync()
}

func (grp *Group) StopSync() error {
	group_log.Debugf("<%s> StopSync called", grp.Item.GroupId)
	grp.ChainCtx.StopSync()
	return nil
}
