package chain

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	pubsubconn "github.com/rumsystem/quorum/internal/pkg/pubsubconn"
	"google.golang.org/protobuf/proto"

	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
)

const (
	Hours = 0
	Mins  = 5
	Sec   = 0
)

const OBJECT_SIZE_LIMIT = 200 * 1024 //(200Kb)

type TrxMgr struct {
	nodename  string
	groupItem *quorumpb.GroupItem
	psconn    pubsubconn.PubSubConn
	groupId   string
}

var trxmgr_log = logging.Logger("trxmgr")

func (trxMgr *TrxMgr) Init(groupItem *quorumpb.GroupItem, psconn pubsubconn.PubSubConn) {
	trxMgr.groupItem = groupItem
	trxMgr.psconn = psconn
	trxMgr.groupId = groupItem.GroupId
	trxmgr_log.Debugf("<%s> trxMgr inited", trxMgr.groupId)
}

func (trxMgr *TrxMgr) SetNodeName(nodename string) {
	trxMgr.nodename = nodename
}

func (trxMgr *TrxMgr) LeaveChannel(cId string) {
	trxMgr.psconn.LeaveChannel(cId)
}

func (trxMgr *TrxMgr) CreateTrxWithoutSign(msgType quorumpb.TrxType, data []byte, encryptto ...[]string) (*quorumpb.Trx, []byte, error) {
	var trx quorumpb.Trx

	trxId := guuid.New()
	trx.TrxId = trxId.String()
	trx.Type = msgType
	trx.GroupId = trxMgr.groupItem.GroupId
	trx.SenderPubkey = trxMgr.groupItem.UserSignPubkey

	var encryptdData []byte

	if msgType == quorumpb.TrxType_POST && trxMgr.groupItem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
		//for post, private group, encrypted by age for all announced group users
		if len(encryptto) == 1 {
			var err error
			ks := localcrypto.GetKeystore()
			if len(encryptto[0]) == 0 {
				return &trx, []byte(""), fmt.Errorf("must have encrypt pubkeys for private group %g", trxMgr.groupItem.GroupId)
			}
			encryptdData, err = ks.EncryptTo(encryptto[0], data)
			if err != nil {
				return &trx, []byte(""), err
			}

		} else {
			return &trx, []byte(""), fmt.Errorf("must have encrypt pubkeys for private group %g", trxMgr.groupItem.GroupId)
		}

	} else {
		var err error
		ciperKey, err := hex.DecodeString(trxMgr.groupItem.CipherKey)
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
	trx.Version = nodectx.GetNodeCtx().Version
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

func (trxMgr *TrxMgr) CreateTrx(msgType quorumpb.TrxType, data []byte, encryptto ...[]string) (*quorumpb.Trx, error) {

	trx, hashed, err := trxMgr.CreateTrxWithoutSign(msgType, data, encryptto...)
	if err != nil {
		return trx, err
	}
	ks := nodectx.GetNodeCtx().Keystore
	keyname := trxMgr.groupItem.GroupId
	if trxMgr.nodename != "" {
		keyname = fmt.Sprintf("%s_%s", trxMgr.nodename, trxMgr.groupItem.GroupId)
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
	trxmgr_log.Debugf("<%s> SendUpdAuthTrx called", trxMgr.groupId)

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
	trxmgr_log.Debugf("<%s> SendRegProducerTrx called", trxMgr.groupId)
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

func (trxMgr *TrxMgr) SendRegUserTrx(item *quorumpb.UserItem) (string, error) {
	trxmgr_log.Debugf("<%s> SendRegUserTrx called", trxMgr.groupId)
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return "", err
	}
	trx, err := trxMgr.CreateTrx(quorumpb.TrxType_USER, encodedcontent)
	err = trxMgr.sendTrx(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	return trx.TrxId, nil
}

func (trxMgr *TrxMgr) SendAnnounceTrx(item *quorumpb.AnnounceItem) (string, error) {
	trxmgr_log.Debugf("<%s> SendAnnounceTrx called", trxMgr.groupId)
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
	trxmgr_log.Debugf("<%s> SendUpdSchemaTrx called", trxMgr.groupId)
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
	trxmgr_log.Debugf("<%s> SendReqBlockResp called", trxMgr.groupId)

	var reqBlockRespItem quorumpb.ReqBlockResp
	reqBlockRespItem.Result = result
	reqBlockRespItem.ProviderPubkey = trxMgr.groupItem.UserSignPubkey
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
	trxmgr_log.Debugf("<%s> SendReqBlockForward called", trxMgr.groupId)

	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = trxMgr.groupItem.UserSignPubkey

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
	trxmgr_log.Debugf("<%s> SendReqBlockBackward called", trxMgr.groupId)

	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = block.BlockId
	reqBlockItem.GroupId = block.GroupId
	reqBlockItem.UserId = trxMgr.groupItem.UserSignPubkey

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
	trxmgr_log.Debugf("<%s> SendBlockProduced called", trxMgr.groupId)
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

func (trxMgr *TrxMgr) PostBytes(trxtype quorumpb.TrxType, encodedcontent []byte, encryptto ...[]string) (string, error) {
	trxmgr_log.Debugf("<%s> PostBytes called", trxMgr.groupId)
	trx, err := trxMgr.CreateTrx(trxtype, encodedcontent, encryptto...)
	err = trxMgr.sendTrx(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	return trx.TrxId, nil
}

func (trxMgr *TrxMgr) PostAny(content proto.Message, encryptto ...[]string) (string, error) {
	trxmgr_log.Debugf("<%s> PostAny called", trxMgr.groupId)

	encodedcontent, err := quorumpb.ContentToBytes(content)
	if err != nil {
		return "", err
	}

	trxmgr_log.Debugf("<%s> content size <%d>", trxMgr.groupId, binary.Size(encodedcontent))
	if binary.Size(encodedcontent) > OBJECT_SIZE_LIMIT {
		err := errors.New("Content size over 200Kb")
		return "", err
	}

	return trxMgr.PostBytes(quorumpb.TrxType_POST, encodedcontent, encryptto...)
}

func (trxMgr *TrxMgr) ResendTrx(trx *quorumpb.Trx) error {
	trxmgr_log.Debugf("<%s> ResendTrx called", trxMgr.groupId)
	return trxMgr.sendTrx(trx)
}

func (trxMgr *TrxMgr) CustomSendTrx(trx *quorumpb.Trx) error {
	trxmgr_log.Debugf("<%s> CustomSendTrx called", trxMgr.groupId)
	return trxMgr.sendTrx(trx)
}

func (trxMgr *TrxMgr) SendBlock(blk *quorumpb.Block) error {
	trxmgr_log.Debugf("<%s> SendBlock called", trxMgr.groupId)

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
	trxmgr_log.Debugf("<%s> sendTrx called", trxMgr.groupId)
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
