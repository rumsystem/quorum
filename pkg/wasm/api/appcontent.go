//go:build js && wasm
// +build js,wasm

package api

import (
	"encoding/hex"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"github.com/rumsystem/quorum/internal/pkg/storage"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
	"google.golang.org/protobuf/proto"
)

type GroupContent struct {
	TrxId     string
	Publisher string
	Content   proto.Message
	TypeUrl   string
	TimeStamp int64
}

type GroupContentResp struct {
	Data *[]GroupContent `json:"data"`
}

func GetContent(groupId string, num int, startTrx string, nonce int64, reverse bool, starttrxinclude bool, senders []string) (*GroupContentResp, error) {
	data := []GroupContent{}

	wasmCtx := quorumContext.GetWASMContext()
	trxids, err := wasmCtx.AppDb.GetGroupContentBySenders(groupId, senders, startTrx, nonce, num, reverse, starttrxinclude)
	if err != nil {
		return nil, err
	}

	groupmgr := chain.GetGroupMgr()
	groupitem, err := groupmgr.GetGroupItem(groupId)
	if err != nil {
		return nil, err
	}
	for _, trxid := range trxids {
		trx, _, err := wasmCtx.DbMgr.GetTrx(trxid.TrxId, storage.Chain, nodectx.GetNodeCtx().Name)
		if err != nil {
			println(err)
			continue
		}

		//decrypt trx data
		if trx.Type == quorumpb.TrxType_POST && groupitem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by pgp for all announced group user
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(groupitem.UserEncryptPubkey, trx.Data)
			if err != nil {
				return nil, err
			}
			trx.Data = decryptData
		} else {
			//decode trx data
			ciperKey, err := hex.DecodeString(groupitem.CipherKey)
			if err != nil {
				return nil, err
			}

			decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
			if err != nil {
				return nil, err
			}
			trx.Data = decryptData
		}

		ctnobj, typeurl, errum := quorumpb.BytesToMessage(trx.TrxId, trx.Data)
		if errum != nil {
			println("Unmarshal trx.Data %s Err: %s", trx.TrxId, errum)
		}
		item := GroupContent{TrxId: trx.TrxId, Publisher: trx.SenderPubkey, Content: ctnobj, TimeStamp: trx.TimeStamp, TypeUrl: typeurl}
		data = append(data, item)
	}

	ret := GroupContentResp{&data}

	return &ret, nil
}
