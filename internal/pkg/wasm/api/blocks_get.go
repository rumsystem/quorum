//go:build js && wasm
// +build js,wasm

package api

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/chain"
	qCrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type DecodeTrxStruct struct {
	Msg     *proto.Message `json:"message"`
	TypeUrl string         `json:"url"`
	TrxId   string         `json:"id"`
}

type DecodeBlockRespStruct struct {
	Trxs  *[]DecodeTrxStruct `json:"trxs"`
	Block *pb.Block          `json:"block"`
}

func GetBlockById(gid string, bid string) (block *pb.Block, err error) {
	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[gid]; ok {
		block, err := group.GetBlock(bid)
		if err != nil {
			return nil, err
		}
		return block, nil
	} else {
		return nil, errors.New(fmt.Sprintf("Group %s not exist", gid))
	}
}

func GetDecodedBlockById(gid string, bid string) (*DecodeBlockRespStruct, error) {
	groupmgr := chain.GetGroupMgr()
	decodedTrxs := &[]DecodeTrxStruct{}
	if group, ok := groupmgr.Groups[gid]; ok {
		block, err := group.GetBlock(bid)
		if err != nil {
			return nil, err
		}
		if group.Item.EncryptType == pb.GroupEncryptType_PUBLIC {
			// aes decode trx
			decodedTrxs, err = aesDecodeTrxData(block, group.Item.CipherKey)
			if err != nil {
				return nil, errors.New(fmt.Sprint("Failed to decode trx: ", err.Error()))
			}
		}

		ret := DecodeBlockRespStruct{decodedTrxs, block}
		return &ret, nil
	} else {
		return nil, errors.New(fmt.Sprintf("Group %s not exist", gid))
	}
}

func aesDecodeTrxData(block *pb.Block, kStr string) (*[]DecodeTrxStruct, error) {
	ret := &[]DecodeTrxStruct{}
	k, err := hex.DecodeString(kStr)
	if err != nil {
		return nil, err
	}
	for _, trx := range block.Trxs {
		decodedData, err := qCrypto.AesDecode(trx.Data, k)
		if err != nil {
			return nil, err
		}
		msg, typeUrl, err := pb.BytesToMessage(trx.TrxId, decodedData)
		if err != nil {
			return nil, err
		}
		s := DecodeTrxStruct{&msg, typeUrl, trx.TrxId}
		*ret = append(*ret, s)
	}
	return ret, nil
}
