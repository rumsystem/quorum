package api

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

type DenyListParam struct {
	Action  string `from:"action" json:"action" validate:"required,oneof=add del"`
	PeerId  string `from:"peer_id"      json:"peer_id"      validate:"required"`
	GroupId string `from:"group_id"  json:"group_id"  validate:"required"`
	Memo    string `from:"memo"  json:"memo"  `
}

type DenyUserResult struct {
	GroupId          string `json:"group_id"`
	PeerId           string `json:"peer_id"`
	GroupOwnerPubkey string `json:"owner_pubkey"`
	Sign             string `json:"sign"`
	TrxId            string `json:"trx_id"`
	Action           string `json:"action"`
	Memo             string `json:"memo"`
}

// @Tags Management
// @Summary DeniedList
// @Description add or remove a user from the denied list
// @Accept json
// @Produce json
// @Param data body DenyListParam true "DenyListParam"
// @Success 200 {object} DenyUserResult
// @Router /api/v1/deniedlist [post]
func (h *Handler) MgrGrpBlkList(c echo.Context) (err error) {

	output := make(map[string]string)
	validate := validator.New()
	params := new(DenyListParam)

	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	var groupSignPubkey []byte
	ks := nodectx.GetNodeCtx().Keystore
	dirks, ok := ks.(*localcrypto.DirKeyStore)
	if ok == true {
		_, err := dirks.GetKeyFromUnlocked(localcrypto.Sign.NameString(params.GroupId))
		if err != nil {
			output[ERROR_INFO] = "group key can't be decoded, err:" + err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		hexkey, err := dirks.GetEncodedPubkey(params.GroupId, localcrypto.Sign)
		pubkeybytes, err := hex.DecodeString(hexkey)
		p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
		groupSignPubkey, err = p2pcrypto.MarshalPublicKey(p2ppubkey)
		if err != nil {
			output[ERROR_INFO] = "group key can't be decoded, err:" + err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
	}

	item := &quorumpb.DenyUserItem{}
	item.GroupId = params.GroupId
	item.PeerId = params.PeerId
	item.GroupOwnerPubkey = p2pcrypto.ConfigEncodeKey(groupSignPubkey)
	item.Action = params.Action
	item.Memo = params.Memo

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[item.GroupId]; !ok {
		output[ERROR_INFO] = "Can not find group"
		return c.JSON(http.StatusBadRequest, output)
	} else if group.Item.OwnerPubKey != group.Item.UserSignPubkey {
		output[ERROR_INFO] = "Only group owner can add or remove user to blocklist"
		return c.JSON(http.StatusBadRequest, output)
	} else {
		var buffer bytes.Buffer
		buffer.Write([]byte(item.GroupId))
		buffer.Write([]byte(item.PeerId))
		buffer.Write(groupSignPubkey)
		buffer.Write([]byte(item.Action))
		buffer.Write([]byte(item.Memo))
		hash := chain.Hash(buffer.Bytes())

		signature, err := ks.SignByKeyName(item.GroupId, hash)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		item.GroupOwnerSign = hex.EncodeToString(signature)
		item.TimeStamp = time.Now().UnixNano()
		trxId, err := group.UpdBlkList(item)

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		blockGrpUserResult := &DenyUserResult{GroupId: item.GroupId, PeerId: item.PeerId, GroupOwnerPubkey: p2pcrypto.ConfigEncodeKey(groupSignPubkey), Sign: hex.EncodeToString(signature), Action: item.Action, Memo: item.Memo, TrxId: trxId}

		return c.JSON(http.StatusOK, blockGrpUserResult)
	}
}
