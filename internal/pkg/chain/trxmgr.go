package chain

import (
	"encoding/hex"
	"fmt"
	"time"

	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	pubsubconn "github.com/rumsystem/quorum/internal/pkg/pubsubconn"
	logging "github.com/ipfs/go-log/v2"
	"google.golang.org/protobuf/proto"

	guuid "github.com/google/uuid"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

const (
	Hours = 0
	Mins  = 5
	Sec   = 0
)

type TrxMgr struct {
	nodename string
	group    *Group
	psconn   pubsubconn.PubSubConn
}

var trxmgr_log = logging.Logger("trx_mgr")

func (trxMgr *TrxMgr) Init(grp *Group, psconn pubsubconn.PubSubConn) {
	trxMgr.group = grp
	trxMgr.psconn = psconn
}

func (trxMgr *TrxMgr) SetNodeName(nodename string) {
	trxMgr.nodename = nodename
}

func (trxMgr *TrxMgr) CreateTrxWithoutSign(msgType quorumpb.TrxType, data []byte) (*quorumpb.Trx, []byte, error) {
	var trx quorumpb.Trx

	trxId := guuid.New()
	trx.TrxId = trxId.String()
	trx.Type = msgType
	trx.GroupId = trxMgr.group.Item.GroupId
	trx.SenderPubkey = trxMgr.group.Item.UserSignPubkey

	var encryptdData []byte

	if msgType == quorumpb.TrxType_POST && trxMgr.group.Item.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		//for post, private group, encrypted by age for all announced group users
		var err error
		announcedUser, err := GetDbMgr().GetAnnouncedUsers(trxMgr.group.Item.GroupId)

		var pubkeys []string
		for _, item := range announcedUser {
			pubkeys = append(pubkeys, item.AnnouncedPubkey)
		}

		ks := localcrypto.GetKeystore()
		encryptdData, err = ks.EncryptTo(pubkeys, data)
		if err != nil {
			return &trx, []byte(""), err
		}
	} else {
		var err error
		ciperKey, err := hex.DecodeString(trxMgr.group.Item.CipherKey)
		if err != nil {
			return &trx, []byte(""), err
		}
		encryptdData, err = localcrypto.AesEncrypt(data, ciperKey)
		if err != nil {
			return &trx, []byte(""), err
		}
	}

	trx.Data = encryptdData

	trx.TimeStamp = time.Now().UnixNano()
	trx.Version = GetNodeCtx().Version
	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))
	trx.Expired = timein.UnixNano()

	bytes, err := proto.Marshal(&trx)
	if err != nil {
		return &trx, []byte(""), err
	}
	hashed := localcrypto.Hash(bytes)
	return &trx, hashed, nil
}

func (trxMgr *TrxMgr) CreateTrx(msgType quorumpb.TrxType, data []byte) (*quorumpb.Trx, error) {

	trx, hashed, err := trxMgr.CreateTrxWithoutSign(msgType, data)
	if err != nil {
		return trx, err
	}
	ks := GetNodeCtx().Keystore
	keyname := trxMgr.group.Item.GroupId
	if trxMgr.nodename != "" {
		keyname = fmt.Sprintf("%s_%s", trxMgr.nodename, trxMgr.group.Item.GroupId)
	}
	signature, err := ks.SignByKeyName(keyname, hashed)

	if err != nil {
		return trx, err
	}

	trx.SenderSign = signature

	return trx, nil
}

func (trxMgr *TrxMgr) VerifyTrx(trx *quorumpb.Trx) (bool, error) {
	//clone trxMsg to verify
	clonetrxmsg := &quorumpb.Trx{
		TrxId:        trx.TrxId,
		Type:         trx.Type,
		GroupId:      trx.GroupId,
		SenderPubkey: trx.SenderPubkey,
		Data:         trx.Data,
		TimeStamp:    trx.TimeStamp,
		Version:      trx.Version,
		Expired:      trx.Expired}

	bytes, err := proto.Marshal(clonetrxmsg)
	if err != nil {
		return false, err
	}

	hashed := localcrypto.Hash(bytes)

	//create pubkey
	serializedpub, err := p2pcrypto.ConfigDecodeKey(trx.SenderPubkey)
	if err != nil {
		return false, err
	}

	pubkey, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
	if err != nil {
		return false, err
	}

	verify, err := pubkey.Verify(hashed, trx.SenderSign)
	return verify, err
}

func (trxMgr *TrxMgr) SendUpdAuthTrx(item *quorumpb.DenyUserItem) (string, error) {
	trxmgr_log.Infof("Send UPD AUTH Trx")

	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return "", err
	}

	trx, err := trxMgr.CreateTrx(quorumpb.TrxType_AUTH, encodedcontent)
	err = trxMgr.sendTrx(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	return trx.TrxId, nil
}

func (trxMgr *TrxMgr) SendRegProducerTrx(item *quorumpb.ProducerItem) (string, error) {
	trxmgr_log.Infof("Send Reg Producer Trx")
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return "", err
	}
	trx, err := trxMgr.CreateTrx(quorumpb.TrxType_PRODUCER, encodedcontent)
	err = trxMgr.sendTrx(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	return trx.TrxId, nil
}

func (trxMgr *TrxMgr) SendAnnounceTrx(item *quorumpb.AnnounceItem) (string, error) {
	trxmgr_log.Infof("Send Announce Trx")
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return "", err
	}

	trx, err := trxMgr.CreateTrx(quorumpb.TrxType_ANNOUNCE, encodedcontent)
	err = trxMgr.sendTrx(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	return trx.TrxId, nil
}

func (trxMgr *TrxMgr) SendUpdSchemaTrx(item *quorumpb.SchemaItem) (string, error) {
	trxmgr_log.Infof("Send Upd Schema Trx")
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return "", err
	}

	trx, err := trxMgr.CreateTrx(quorumpb.TrxType_SCHEMA, encodedcontent)
	err = trxMgr.sendTrx(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	return trx.TrxId, nil
}

func (trxMgr *TrxMgr) SendReqBlockResp(req *quorumpb.ReqBlock, block *quorumpb.Block, result quorumpb.ReqBlkResult) error {
	chain_log.Infof("SendReqBlockResp called")

	var reqBlockRespItem quorumpb.ReqBlockResp
	reqBlockRespItem.Result = result
	reqBlockRespItem.ProviderPubkey = trxMgr.group.Item.UserSignPubkey
	reqBlockRespItem.RequesterPubkey = req.UserId
	reqBlockRespItem.GroupId = req.GroupId
	reqBlockRespItem.BlockId = req.BlockId

	pbBytesBlock, err := proto.Marshal(block)
	if err != nil {
		return err
	}
	reqBlockRespItem.Block = pbBytesBlock

	bItemBytes, err := proto.Marshal(&reqBlockRespItem)
	if err != nil {
		return err
	}

	//send ask next block trx out
	trx, err := trxMgr.CreateTrx(quorumpb.TrxType_REQ_BLOCK_RESP, bItemBytes)
	if err != nil {
		trxmgr_log.Warningf(err.Error())
		return err
	}

	return trxMgr.sendTrx(trx)
}

func (trxMgr *TrxMgr) SendReqBlockForward(block *quorumpb.Block) error {
	trxmgr_log.Infof("SendReqBlock called")

	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = trxMgr.group.Item.UserSignPubkey

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		group_log.Warningf(err.Error())
		return err
	}

	trx, err := trxMgr.CreateTrx(quorumpb.TrxType_REQ_BLOCK_FORWARD, bItemBytes)
	if err != nil {
		group_log.Warningf(err.Error())
		return err
	}

	return trxMgr.sendTrx(trx)
}

func (trxMgr *TrxMgr) SendReqBlockBackward(block *quorumpb.Block) error {
	trxmgr_log.Infof("SendReqBlock called")

	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = trxMgr.group.Item.UserSignPubkey

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		group_log.Warningf(err.Error())
		return err
	}

	trx, err := trxMgr.CreateTrx(quorumpb.TrxType_REQ_BLOCK_BACKWARD, bItemBytes)
	if err != nil {
		group_log.Warningf(err.Error())
		return err
	}

	return trxMgr.sendTrx(trx)
}

func (trxMgr *TrxMgr) SendBlockProduced(blk *quorumpb.Block) error {
	trxmgr_log.Infof("SendBlockProduced called")
	encodedcontent, err := proto.Marshal(blk)
	if err != nil {
		return err
	}
	trx, err := trxMgr.CreateTrx(quorumpb.TrxType_BLOCK_PRODUCED, encodedcontent)
	if err != nil {
		return err
	}
	return trxMgr.sendTrx(trx)
}

func (trxMgr *TrxMgr) PostBytes(trxtype quorumpb.TrxType, encodedcontent []byte) (string, error) {
	trxmgr_log.Infof("PostBytes called")
	trx, err := trxMgr.CreateTrx(trxtype, encodedcontent)
	err = trxMgr.sendTrx(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	return trx.TrxId, nil
}

func (trxMgr *TrxMgr) PostAny(content proto.Message) (string, error) {
	trxmgr_log.Infof("PostAny called")
	encodedcontent, err := quorumpb.ContentToBytes(content)
	if err != nil {
		return "", err
	}
	return trxMgr.PostBytes(quorumpb.TrxType_POST, encodedcontent)
}

func (trxMgr *TrxMgr) ResendTrx(trx *quorumpb.Trx) error {
	return trxMgr.sendTrx(trx)
}

func (trxMgr *TrxMgr) CustomSendTrx(trx *quorumpb.Trx) error {
	return trxMgr.sendTrx(trx)
}

func (trxMgr *TrxMgr) SendBlock(blk *quorumpb.Block) error {
	trxmgr_log.Infof("SendBlock called")

	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(blk)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_BLOCK
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	return trxMgr.psconn.Publish(pkgBytes)
}

func (trxMgr *TrxMgr) sendTrx(trx *quorumpb.Trx) error {
	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(trx)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_TRX
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	return trxMgr.psconn.Publish(pkgBytes)
}
