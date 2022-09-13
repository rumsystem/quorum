package chain

import (
	"bytes"
	"encoding/hex"
	"time"

	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/conn"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
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

func (grp *Group) LoadGroup(item *quorumpb.GroupItem) {
	group_log.Debugf("<%s> Init called", item.GroupId)
	grp.Item = item

	//create and initial chain
	grp.ChainCtx = &Chain{}
	grp.ChainCtx.ChainInit(grp)

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
		group_log.Debugf("<%s> Private group load announced user key", item.GroupId)
		grp.ChainCtx.UpdUserList()
	}

	//register chainctx with conn
	conn.GetConn().RegisterChainCtx(item.GroupId, item.OwnerPubKey, item.UserSignPubkey, grp.ChainCtx)
	//reload producers
	grp.ChainCtx.UpdProducerList()
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
	grp.ChainCtx.ChainInit(grp)

	err := nodectx.GetNodeCtx().GetChainStorage().AddGensisBlock(item.GenesisBlock, grp.ChainCtx.nodename)
	if err != nil {
		return err
	}

	group_log.Debugf("<%s> Update nonce called, with nodename <%s>", item.GroupId, grp.ChainCtx.nodename)

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
	hash := localcrypto.Hash(buffer.Bytes())

	ks := nodectx.GetNodeCtx().Keystore
	signature, err := ks.EthSignByKeyName(item.GroupId, hash)
	if err != nil {
		return err
	}

	pItem.GroupOwnerSign = hex.EncodeToString(signature)
	pItem.Memo = "Owner Registated as the first oroducer"
	pItem.TimeStamp = time.Now().UnixNano()

	err = nodectx.GetNodeCtx().GetChainStorage().AddProducer(pItem, grp.ChainCtx.nodename)
	if err != nil {
		return err
	}

	group_log.Infof("Group <%s> created", grp.Item.GroupId)

	err = nodectx.GetNodeCtx().GetChainStorage().AddGroup(grp.Item)
	if err != nil {
		return err
	}

	//register chainctx with conn
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
	return nodectx.GetNodeCtx().GetChainStorage().RmGroup(grp.Item.GroupId)
}

func (grp *Group) ClearGroup() error {
	return nodectx.GetNodeCtx().GetChainStorage().RemoveGroupData(grp.Item, grp.ChainCtx.nodename)
}

func (grp *Group) StartSync(restart bool) error {
	group_log.Debugf("<%s> StartSync called", grp.Item.GroupId)
	if restart == true {
		grp.ChainCtx.StopSync()
	}
	//time.Sleep(10 * time.Second)
	return grp.ChainCtx.StartSync()
	//return grp.ChainCtx.SyncForward(grp.ChainCtx.group.Item.HighestBlockId, grp.ChainCtx.nodename)
}

func (grp *Group) StopSync() error {
	group_log.Debugf("<%s> StopSync called", grp.Item.GroupId)
	grp.ChainCtx.StopSync()
	return nil
}

func (grp *Group) GetSyncerStatus() int8 {
	return grp.ChainCtx.GetSyncStatus()
	return 0
}

func (grp *Group) GetSnapshotInfo() (tag *quorumpb.SnapShotTag, err error) {
	return grp.ChainCtx.GetSnapshotTag()
}

func (grp *Group) GetBlock(blockId string) (*quorumpb.Block, error) {
	group_log.Debugf("<%s> GetBlock called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetBlock(blockId, false, grp.ChainCtx.nodename)
}

func (grp *Group) GetTrx(trxId string) (*quorumpb.Trx, []int64, error) {
	group_log.Debugf("<%s> GetTrx called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetTrx(trxId, def.Chain, grp.ChainCtx.nodename)
}

func (grp *Group) GetTrxFromCache(trxId string) (*quorumpb.Trx, []int64, error) {
	group_log.Debugf("<%s> GetTrx called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetTrx(trxId, def.Cache, grp.ChainCtx.nodename)
}

//func (grp *Group) GetProducers() ([]*quorumpb.ProducerItem, error) {
//	group_log.Debugf("<%s> GetProducers called", grp.Item.GroupId)
//	return nodectx.GetNodeCtx().GetChainStorage().GetProducers(grp.Item.GroupId, grp.ChainCtx.nodename)
//}

func (grp *Group) GetSchemas() ([]*quorumpb.SchemaItem, error) {
	group_log.Debugf("<%s> GetSchema called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAllSchemasByGroup(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetAnnouncedProducer(pubkey string) (*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedProducer called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAnnouncedProducer(grp.Item.GroupId, pubkey, grp.ChainCtx.nodename)
}

func (grp *Group) GetAnnouncedUser(pubkey string) (*quorumpb.AnnounceItem, error) {
	group_log.Debugf("<%s> GetAnnouncedUser called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAnnouncedUser(grp.Item.GroupId, pubkey, grp.ChainCtx.nodename)
}

func (grp *Group) GetAppConfigKeyList() (keyName []string, itemType []string, err error) {
	group_log.Debugf("<%s> GetAppConfigKeyList called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigKey(grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) GetAppConfigItem(keyName string) (*quorumpb.AppConfigItem, error) {
	group_log.Debugf("<%s> GetAppConfigItem called", grp.Item.GroupId)
	return nodectx.GetNodeCtx().GetChainStorage().GetAppConfigItem(keyName, grp.Item.GroupId, grp.ChainCtx.nodename)
}

func (grp *Group) UpdAnnounce(item *quorumpb.AnnounceItem) (string, error) {
	group_log.Debugf("<%s> UpdAnnounce called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetAnnounceTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx, conn.ProducerChannel)
}

func (grp *Group) PostToGroup(content proto.Message) (string, error) {
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
		return grp.sendTrx(trx, conn.ProducerChannel)
	}
	trx, err := grp.ChainCtx.GetTrxFactory().GetPostAnyTrx("", content)
	if err != nil {
		return "", err
	}
	return grp.sendTrx(trx, conn.ProducerChannel)
}

func (grp *Group) UpdProducer(item *quorumpb.ProducerItem) (string, error) {
	group_log.Debugf("<%s> UpdProducer called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetRegProducerTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx, conn.ProducerChannel)
}

func (grp *Group) UpdUser(item *quorumpb.UserItem) (string, error) {
	group_log.Debugf("<%s> UpdUser called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetRegUserTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx, conn.ProducerChannel)
}

func (grp *Group) UpdSchema(item *quorumpb.SchemaItem) (string, error) {
	group_log.Debugf("<%s> UpdSchema called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdSchemaTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx, conn.ProducerChannel)
}

func (grp *Group) UpdAppConfig(item *quorumpb.AppConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdAppConfig called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetUpdAppConfigTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx, conn.ProducerChannel)
}

func (grp *Group) SendRawTrx(trx *quorumpb.Trx) (string, error) {
	return grp.sendTrx(trx, conn.ProducerChannel)
}

func (grp *Group) sendTrx(trx *quorumpb.Trx, channel conn.PsConnChanel) (string, error) {
	connMgr, err := conn.GetConn().GetConnMgr(grp.Item.GroupId)
	if err != nil {
		return "", err
	}
	err = connMgr.SendTrxPubsub(trx, channel)
	if err != nil {
		return "", err
	}

	err = grp.ChainCtx.GetPubqueueIface().TrxEnqueue(grp.Item.GroupId, trx)
	if err != nil {
		return "", err
	}

	return trx.TrxId, nil
}

func (grp *Group) UpdChainConfig(item *quorumpb.ChainConfigItem) (string, error) {
	group_log.Debugf("<%s> UpdChainSendTrxRule called", grp.Item.GroupId)
	trx, err := grp.ChainCtx.GetTrxFactory().GetChainConfigTrx("", item)
	if err != nil {
		return "", nil
	}
	return grp.sendTrx(trx, conn.ProducerChannel)
}
