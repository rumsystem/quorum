package chain

import (
	"crypto/sha256"
	//"encoding/hex"
	"encoding/json"
	"time"

	guuid "github.com/google/uuid"
)

//transaction will expire in 1 hour (after send)
const Hours = 1
const Mins = 0
const Sec = 0

// transaction status
type TrxType int8

const (
	REQ_SIGN            TrxType = 0 // request witness sign
	REQ_SIGN_RESP       TrxType = 1 // response requset withness sign
	ADD_NEW_BLOCK       TrxType = 2 // new block created
	ADD_NEW_BLOCK_RESP  TrxType = 3 // response new block created
	REQ_NEXT_BLOCK      TrxType = 4 // request next block
	REQ_NEXT_BLOCK_RESP TrxType = 5 // response request next block
	PEER_ANNOUNCE       TrxType = 6 // announce I am online
)

type ReqBlock int8

const (
	BLOCK_IN_TRX ReqBlock = 0 //block data in trx
	BLOCK_ON_TOP ReqBlock = 1 //top block, no new block, block in trx is empty
)

type Trx struct {
	Msg       TrxMsg
	Data      []byte
	Consensus []string
}

type TrxMsg struct {
	TrxId     string
	MsgType   TrxType
	Sender    string
	GroupId   string
	Data      []byte
	Version   string
	TimeStamp int64
}

type ReqSign struct {
	Hash       []byte
	Signature  []byte
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

func CreateTrxMsgReqSign(groupId string, data []byte) (TrxMsg, error) {
	var trxMsg TrxMsg
	var reqSign ReqSign

	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))

	reqSign.Expiration = timein.String()
	reqSign.Hash = hash(data)
	reqSign.Signature = []byte("Signature of original data (signed by using peer private key")

	trxId := guuid.New()
	payload, _ := json.Marshal(reqSign)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = REQ_SIGN
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = groupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	return trxMsg, nil
}

func CreateTrxMsgReqSignResp(inTrxMsg TrxMsg, reqSign ReqSign) (TrxMsg, error) {
	var trxMsg TrxMsg
	var respSign ReqSignResp

	respSign.ReqTrxId = inTrxMsg.TrxId
	respSign.Requester = inTrxMsg.Sender
	respSign.Witness = GetChainCtx().PeerId.Pretty()
	respSign.Hash = []byte("Hash generated by signature provider")
	respSign.WitnessSign = []byte("Signed by " + GetChainCtx().PeerId.Pretty())

	trxId := guuid.New()
	payload, _ := json.Marshal(respSign)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = REQ_SIGN_RESP
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = inTrxMsg.GroupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	return trxMsg, nil
}

func CreateTrxNewBlock(block Block) (TrxMsg, error) {
	var trxMsg TrxMsg
	var newBlock NewBlock

	newBlock.Producer = GetChainCtx().PeerId.Pretty()
	newBlock.BlockId = block.Cid

	payloadblock, _ := json.Marshal(block)
	newBlock.Data = payloadblock

	trxId := guuid.New()
	payloadmsg, _ := json.Marshal(newBlock)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = ADD_NEW_BLOCK
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payloadmsg
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	return trxMsg, nil
}

func CreateTrxNewBlockResp(block Block) (TrxMsg, error) {
	var trxMsg TrxMsg
	var newBlockResp NewBlockResp

	newBlockResp.Producer = block.Producer
	newBlockResp.BlockId = block.Cid
	newBlockResp.StorageProvider = GetChainCtx().PeerId.Pretty()

	trxId := guuid.New()
	payload, _ := json.Marshal(newBlockResp)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = ADD_NEW_BLOCK_RESP
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	return trxMsg, nil
}

func CreateTrxReqNextBlock(block Block) (TrxMsg, error) {
	var trxMsg TrxMsg
	var reqNextBlock ReqNextBlock

	reqNextBlock.BlockId = block.Cid

	trxId := guuid.New()
	payload, _ := json.Marshal(reqNextBlock)

	trxMsg.TrxId = trxId.String()
	trxMsg.MsgType = REQ_NEXT_BLOCK
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payload
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	return trxMsg, nil
}

func CreateTrxReqNextBlockResp(resp ReqBlock, requester string, block Block) (TrxMsg, error) {
	var trxMsg TrxMsg
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
	trxMsg.MsgType = REQ_NEXT_BLOCK_RESP
	trxMsg.Sender = GetChainCtx().PeerId.Pretty()
	trxMsg.GroupId = block.GroupId
	trxMsg.Data = payloadmsg
	trxMsg.Version = GetChainCtx().Version
	trxMsg.TimeStamp = time.Now().UnixNano()

	return trxMsg, nil
}

func CreateTrxPeerAnnounce() (TrxMsg, error) {
	var trxMsg TrxMsg
	return trxMsg, nil
}

func hash(data []byte) []byte {
	h := sha256.New()
	h.Write([]byte(data))
	hashed := h.Sum(nil)
	return hashed
	//return hex.EncodeToString(hashed)
}

//sign trx by using private key
func SignTrx(hash []byte) (singature string, err error) {
	var sig string
	return sig, nil
}

//check if hash is available
func VerifySig(hash []byte, signature string) (bool, error) {
	return true, nil
}
