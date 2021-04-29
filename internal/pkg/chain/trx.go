package chain

import (
	"crypto/sha256"
	"encoding/json"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
	"time"

	//"github.com/golang/glog"
	guuid "github.com/google/uuid"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
)

//transaction will expire in 1 hour (after send)
const Hours = 1
const Mins = 0
const Sec = 0

type ReqBlock int8

const (
	BLOCK_IN_TRX ReqBlock = 0 //block data in trx
	BLOCK_ON_TOP ReqBlock = 1 //top block, no new block, block in trx is empty
)

type Trx struct {
	Msg       quorumpb.TrxMsg
	Data      []byte
	Consensus []string
}

type ReqSign struct {
	Datahash   []byte
	Expiration string
}

type ReqSignResp struct {
	ReqTrxId    string
	Requester   string
	Witness     string
	Hash        []byte
	WitnessSign []byte
}

type NewBlock struct {
	Producer string
	BlockId  string
	Data     []byte //the whole block
}

type NewBlockResp struct {
	Producer        string
	BlockId         string
	StorageProvider string
}

type ReqNextBlock struct {
	BlockId string //block id
}

type ReqNextBlockResp struct {
	Provider  string   //who provide the block
	Requester string   //who request the block
	Response  ReqBlock //response
	BlockId   string
	Block     []byte //the whole block data
}

func CreateTrxMsgReqSign(groupId string, data []byte) (quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var reqSign ReqSign

	trxId := guuid.New()
	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = quorumpb.TrxType_REQ_SIGN
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = groupId
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return trxMsg, err
	}
	trxMsg.Pubkey = pubkey

	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))

	reqSign.Expiration = timein.String()
	reqSign.Datahash = hash(data)

	payload, _ := json.Marshal(reqSign)
	trxMsg.Data = payload

	sign, err := signTrx(trxMsg)

	if err != nil {
		return trxMsg, err
	}

	trxMsg.Sign = sign

	return trxMsg, nil
}

func CreateTrxMsgReqSignResp(inTrxMsg quorumpb.TrxMsg, reqSign ReqSign) (quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var respSign ReqSignResp

	respSign.ReqTrxId = inTrxMsg.TrxId
	respSign.Requester = inTrxMsg.Sender
	respSign.Witness = GetChainCtx().PeerId.Pretty()
	bytes, err := json.Marshal(trxMsg)
	hashed := hash(bytes)
	respSign.Hash = hashed
	sign, err := signTrx(inTrxMsg)
	if err != nil {
		return trxMsg, err
	}
	respSign.WitnessSign = sign

	trxId := guuid.New()
	payload, _ := json.Marshal(respSign)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = quorumpb.TrxType_REQ_SIGN_RESP
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = inTrxMsg.GroupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return trxMsg, err
	}

	trxMsg.Pubkey = pubkey

	sign, err = signTrx(trxMsg)

	if err != nil {
		return trxMsg, err
	}

	trxMsg.Sign = sign

	return trxMsg, nil
}

func CreateTrxNewBlock(block Block) (quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var newBlock NewBlock

	newBlock.Producer = GetChainCtx().PeerId.Pretty()
	newBlock.BlockId = block.Cid

	payloadblock, _ := json.Marshal(block)
	newBlock.Data = payloadblock

	trxId := guuid.New()
	payloadmsg, _ := json.Marshal(newBlock)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = quorumpb.TrxType_ADD_NEW_BLOCK
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payloadmsg
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return trxMsg, err
	}

	trxMsg.Pubkey = pubkey

	sign, err := signTrx(trxMsg)
	if err != nil {
		return trxMsg, err
	}

	trxMsg.Sign = sign
	return trxMsg, nil
}

func CreateTrxNewBlockResp(block Block) (quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var newBlockResp NewBlockResp

	newBlockResp.Producer = block.Producer
	newBlockResp.BlockId = block.Cid
	newBlockResp.StorageProvider = GetChainCtx().PeerId.Pretty()

	trxId := guuid.New()
	payload, _ := json.Marshal(newBlockResp)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = quorumpb.TrxType_ADD_NEW_BLOCK_RESP
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return trxMsg, err
	}
	trxMsg.Pubkey = pubkey

	sign, err := signTrx(trxMsg)

	if err != nil {
		return trxMsg, err
	}

	trxMsg.Sign = sign

	return trxMsg, nil
}

func CreateTrxReqNextBlock(block Block) (quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var reqNextBlock ReqNextBlock

	reqNextBlock.BlockId = block.Cid

	trxId := guuid.New()
	payload, _ := json.Marshal(reqNextBlock)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = quorumpb.TrxType_REQ_NEXT_BLOCK
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	pubkey, err := getPubKey()
	if err != nil {
		return trxMsg, err
	}
	trxMsg.Pubkey = pubkey

	sign, err := signTrx(trxMsg)

	if err != nil {
		return trxMsg, err
	}

	trxMsg.Sign = sign
	return trxMsg, nil
}

func CreateTrxReqNextBlockResp(resp ReqBlock, requester string, block Block) (quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	var respMsg ReqNextBlockResp

	respMsg.Provider = GetChainCtx().PeerId.Pretty()
	respMsg.Requester = requester
	respMsg.Response = resp
	respMsg.BlockId = block.Cid
	payload, err := json.Marshal(block)
	if err != nil {
		return trxMsg, err
	}

	respMsg.Block = payload
	trxId := guuid.New()
	payloadmsg, err := json.Marshal(respMsg)
	if err != nil {
		return trxMsg, err
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
		return trxMsg, err
	}
	trxMsg.Pubkey = pubkey

	sign, err := signTrx(trxMsg)

	if err != nil {
		return trxMsg, err
	}

	trxMsg.Sign = sign

	return trxMsg, nil
}

func CreateTrxPeerAnnounce() (quorumpb.TrxMsg, error) {
	var trxMsg quorumpb.TrxMsg
	return trxMsg, nil
}

func hash(data []byte) []byte {
	h := sha256.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	return hashed
}

func signTrx(trxMsg quorumpb.TrxMsg) ([]byte, error) {
	bytes, err := proto.Marshal(&trxMsg)
	hashed := hash(bytes)
	signature, err := GetChainCtx().Privatekey.Sign(hashed)
	return signature, err
}

func VerifyTrx(trxMsg quorumpb.TrxMsg) (bool, error) {
	//get signature
	sign := trxMsg.Sign
	trxMsg.Sign = nil
	bytes, err := proto.Marshal(&trxMsg)
	if err != nil {
		return false, err
	}
	hashed := hash(bytes)

	//create pubkey
	serializedpub, err := p2pcrypto.ConfigDecodeKey(trxMsg.Pubkey)
	if err != nil {
		return false, err
	}

	pubkey, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
	if err != nil {
		return false, err
	}

	verify, err := pubkey.Verify(hashed, sign)
	return verify, err
}

func getPubKey() (string, error) {
	var pubkey string
	pubkeybytes, err := p2pcrypto.MarshalPublicKey(GetChainCtx().PublicKey)
	if err != nil {
		return pubkey, err
	}

	pubkey = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	return pubkey, nil
}
