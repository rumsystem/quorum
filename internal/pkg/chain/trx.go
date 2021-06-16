package chain

import (
	"crypto/sha256"
	"time"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"

	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

//transaction will expire in 5 minutes (after send)
const Hours = 0
const Mins = 5
const Sec = 0

func CreateTrx(msgType quorumpb.TrxType, groupId string, data []byte) (*quorumpb.Trx, error) {
	var trx quorumpb.Trx

	trxId := guuid.New()
	trx.TrxId = trxId.String()
	trx.Type = msgType
	trx.GroupId = groupId
	trx.Sender = GetChainCtx().PeerId.Pretty()

	pubkey, err := getPubKey()
	if err != nil {
		return &trx, err
	}
	trx.Pubkey = pubkey
	trx.Data = data
	trx.TimeStamp = time.Now().UnixNano()
	trx.Version = GetChainCtx().Version
	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))
	trx.Expired = timein.UnixNano()

	sign, err := signTrx(&trx)

	if err != nil {
		return &trx, err
	}

	trx.Signature = sign

	return &trx, nil
}

/*

func CreateTrxMsgReqSign(msgType quorumpb.TrxType, groupId string, data []byte) (*quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var reqSign quorumpb.ReqSign

	trxId := guuid.New()
	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = msgType // quorumpb.TrxType_REQ_SIGN_POST
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = groupId
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return &trxMsg, err
	}
	trxMsg.Pubkey = pubkey

	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))

	reqSign.Expiration = timein.String()
	reqSign.Datahash = Hash(data)

	payload, _ := proto.Marshal(&reqSign)
	trxMsg.Data = payload

	sign, err := signTrx(&trxMsg)

	if err != nil {
		return &trxMsg, err
	}

	trxMsg.Sign = sign

	return &trxMsg, nil
}

//TOFIX: reqSign *quorumpb.ReqSign unused in this func?
func CreateTrxMsgReqSignResp(inTrxMsg *quorumpb.TrxMsg, reqSign *quorumpb.ReqSign) (*quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var respSign quorumpb.ReqSignResp

	respSign.ReqTrxId = inTrxMsg.TrxId
	respSign.Requester = inTrxMsg.Sender
	respSign.Witness = GetChainCtx().PeerId.Pretty()
	bytes, err := proto.Marshal(inTrxMsg)
	hashed := Hash(bytes)
	respSign.Hash = hashed
	sign, err := signTrx(inTrxMsg)
	if err != nil {
		return &trxMsg, err
	}
	respSign.WitnessSign = sign

	trxId := guuid.New()
	payload, _ := proto.Marshal(&respSign) //TODO: catch proto.Marshal err?

	trxMsg.TrxId = trxId.String()

	if inTrxMsg.MsgType == quorumpb.TrxType_REQ_SIGN_POST {
		trxMsg.MsgType = quorumpb.TrxType_REQ_SIGN_RESP_POST
	} else if inTrxMsg.MsgType == quorumpb.TrxType_REQ_SIGN_AUTH {
		trxMsg.MsgType = quorumpb.TrxType_REQ_SIGN_RESP_AUTH
	} else {
		glog.Warning("Unknown msgType")
		err := errors.New("Unknown msgType")
		return &trxMsg, err
	}

	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = inTrxMsg.GroupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return &trxMsg, err
	}

	trxMsg.Pubkey = pubkey

	sign, err = signTrx(&trxMsg)

	if err != nil {
		return &trxMsg, err
	}

	trxMsg.Sign = sign

	return &trxMsg, nil
}

func CreateTrxNewBlock(block *quorumpb.Block) (*quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var newBlock quorumpb.NewBlock

	newBlock.Producer = GetChainCtx().PeerId.Pretty()
	newBlock.BlockId = block.Cid

	payloadblock, _ := proto.Marshal(block)
	newBlock.Data = payloadblock

	trxId := guuid.New()
	payloadmsg, _ := proto.Marshal(&newBlock)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = quorumpb.TrxType_ADD_NEW_BLOCK
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payloadmsg
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return &trxMsg, err
	}

	trxMsg.Pubkey = pubkey

	sign, err := signTrx(&trxMsg)
	if err != nil {
		return &trxMsg, err
	}

	trxMsg.Sign = sign
	return &trxMsg, nil
}

func CreateTrxNewBlockResp(block *quorumpb.Block) (*quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var newBlockResp quorumpb.NewBlockResp

	newBlockResp.Producer = block.Producer
	newBlockResp.BlockId = block.Cid
	newBlockResp.StorageProvider = GetChainCtx().PeerId.Pretty()

	trxId := guuid.New()
	payload, _ := proto.Marshal(&newBlockResp)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = quorumpb.TrxType_ADD_NEW_BLOCK_RESP
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return &trxMsg, err
	}
	trxMsg.Pubkey = pubkey

	sign, err := signTrx(&trxMsg)

	if err != nil {
		return &trxMsg, err
	}

	trxMsg.Sign = sign

	return &trxMsg, nil
}

func CreateTrxReqNextBlock(block *quorumpb.Block) (*quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var reqNextBlock quorumpb.ReqNextBlock

	reqNextBlock.BlockId = block.Cid

	trxId := guuid.New()
	payload, _ := proto.Marshal(&reqNextBlock)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = quorumpb.TrxType_REQ_NEXT_BLOCK
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return &trxMsg, err
	}
	trxMsg.Pubkey = pubkey

	sign, err := signTrx(&trxMsg)

	if err != nil {
		return &trxMsg, err
	}

	trxMsg.Sign = sign
	return &trxMsg, nil
}

func CreateTrxReqNextBlockResp(resp quorumpb.ReqBlock, requester string, block *quorumpb.Block) (*quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var respMsg quorumpb.ReqNextBlockResp

	respMsg.Provider = GetChainCtx().PeerId.Pretty()
	respMsg.Requester = requester
	respMsg.Response = resp
	respMsg.BlockId = block.Cid
	payload, err := proto.Marshal(block)
	if err != nil {
		return &trxMsg, err
	}

	respMsg.Block = payload
	trxId := guuid.New()
	payloadmsg, err := proto.Marshal(&respMsg)
	if err != nil {
		return &trxMsg, err
	}

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = quorumpb.TrxType_REQ_NEXT_BLOCK_RESP
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payloadmsg
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return &trxMsg, err
	}
	trxMsg.Pubkey = pubkey

	sign, err := signTrx(&trxMsg)

	if err != nil {
		return &trxMsg, err
	}

	trxMsg.Sign = sign

	return &trxMsg, nil
}

func CreateTrxPeerAnnounce() (*quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	return &trxMsg, nil
}

*/
func Hash(data []byte) []byte {
	h := sha256.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	return hashed
}

func signTrx(trx *quorumpb.Trx) ([]byte, error) {
	bytes, err := proto.Marshal(trx)
	hashed := Hash(bytes)
	signature, err := GetChainCtx().Privatekey.Sign(hashed)
	return signature, err
}

func VerifyTrx(trx *quorumpb.Trx) (bool, error) {
	//clone trxMsg to verify
	clonetrxmsg := &quorumpb.Trx{
		TrxId:     trx.TrxId,
		Type:      trx.Type,
		GroupId:   trx.GroupId,
		Sender:    trx.Sender,
		Pubkey:    trx.Pubkey,
		Data:      trx.Data,
		TimeStamp: trx.TimeStamp,
		Version:   trx.Version,
		Expired:   trx.Expired}

	bytes, err := proto.Marshal(clonetrxmsg)
	if err != nil {
		return false, err
	}

	hashed := Hash(bytes)

	//create pubkey
	serializedpub, err := p2pcrypto.ConfigDecodeKey(trx.Pubkey)
	if err != nil {
		return false, err
	}

	pubkey, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
	if err != nil {
		return false, err
	}

	verify, err := pubkey.Verify(hashed, trx.Signature)
	return verify, err
}
