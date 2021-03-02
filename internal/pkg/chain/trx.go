package chain

import (
	"encoding/json"
	guuid "github.com/google/uuid"
	"time"
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
	NEW_BLOCK           TrxType = 2 // new block created
	NEW_BLOCK_RESP      TrxType = 3 // response new block created
	REQ_NEXT_BLOCK      TrxType = 4 // request next block
	REQ_NEXT_BLOCK_RESP TrxType = 5 // response request next block
	PEER_ANNOUNCE       TrxType = 6 // announce I am online
)

type Trx struct {
	Msg       TrxMsg
	Data      []byte
	Consensus []string
}

type TrxMsg struct {
	MsgType TrxType
	TrxId   string
	Data    []byte
}

type TrxRegSign struct {
	Sender     string
	Hash       []byte
	Signature  []byte //sender signature
	Expiration string
}

type TrxReqSignResp struct {
	ReqTrxId  string
	Witness   string //username + publickey
	Hash      []byte //hash of the whole TrxRegSign
	Signature []byte //witness signature
}

type NewBlock struct {
	BlockId string
	Data    []byte
}

type NewBlockResp struct {
	BlockId         string
	StorageProvider string //who "save" the block
	Log             string
}

type ReqNextBlock struct {
	Sender    string //who request the block
	SenderPK  []byte //sender's public key
	GroupId   string //block groupId
	BlockId   string //block id
	Timestamp string
}

type ReqNextBlockResp struct {
	StorageProvider string //who provide the block
	Block           []byte //block data
	Log             string //log
	Timestamp       string
}

func CreateTrxMsgRegSign(sender string, data []byte) (TrxMsg, error) {
	var trxMsg TrxMsg
	var reqSign TrxRegSign

	timein := time.Now().Local().Add(time.Hour*time.Duration(Hours) +
		time.Minute*time.Duration(Mins) +
		time.Second*time.Duration(Sec))

	//hash data
	var hash []byte

	//sign data hash result with private_key
	var sig []byte

	reqSign.Sender = sender
	reqSign.Expiration = timein.String()
	reqSign.Hash = hash
	reqSign.Signature = sig

	trxMsg.MsgType = REQ_SIGN

	//create trxid (uuid)
	trxId := guuid.New()
	trxMsg.TrxId = trxId.String()

	payload, err := json.Marshal(reqSign)
	trxMsg.Data = payload

	return trxMsg, err
}

func CreateTrxMsgRegSignResp(reqTrxId, witness string, regSign TrxRegSign) (TrxMsg, error) {
	var trxMsg TrxMsg
	var respSign TrxReqSignResp

	//hash reqSign with sender signature
	var hash []byte

	//sign reqSign hash result with private_key
	sig := []byte("SIGNED")

	respSign.ReqTrxId = reqTrxId
	respSign.Witness = witness
	respSign.Hash = hash
	respSign.Signature = sig

	trxMsg.MsgType = REQ_SIGN_RESP

	//create trxid (uuid)
	trxId := guuid.New()
	trxMsg.TrxId = trxId.String()

	payload, err := json.Marshal(respSign)
	trxMsg.Data = payload

	return trxMsg, err
}

func CreateTrxNewBlock() (TrxMsg, error) {
	var trxMsg TrxMsg
	return trxMsg, nil
}

func CreateTrxNewBlockResp() (TrxMsg, error) {
	var trxMsg TrxMsg
	return trxMsg, nil
}

func CreateReqNextBlock() (TrxMsg, error) {
	var trxMsg TrxMsg
	return trxMsg, nil
}

func CreateReqNextBlockResp() (TrxMsg, error) {
	var trxMsg TrxMsg
	return trxMsg, nil
}

func CreatePeerAnnounce() (TrxMsg, error) {
	var trxMsg TrxMsg
	return trxMsg, nil
}

func hash(data []byte) []byte {
	var h []byte
	return h
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
