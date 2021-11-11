package api

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
)

type LeaveGroupParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
}

type LeaveGroupResult struct {
	GroupId   string `json:"group_id" validate:"required"`
	Signature string `json:"signature" validate:"required"`
}

// @Tags Groups
// @Summary LeaveGroup
// @Description Leave a new group
// @Accept json
// @Produce json
// @Param data body LeaveGroupParam true "LeaveGroupParam"
// @success 200 {object} LeaveGroupResult "LeaveGroupResult"
// @Router /api/v1/group/leave [post]
func (h *Handler) LeaveGroup(c echo.Context) (err error) {

	output := make(map[string]string)
	validate := validator.New()
	params := new(LeaveGroupParam)

	if err := c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	if group, ok := groupmgr.Groups[params.GroupId]; ok {
		err := group.LeaveGrp()

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		delete(groupmgr.Groups, params.GroupId)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		var groupSignPubkey []byte
		ks := localcrypto.GetKeystore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if ok == true {
			hexkey, err := dirks.GetEncodedPubkey("default", localcrypto.Sign)
			pubkeybytes, err := hex.DecodeString(hexkey)
			p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
			groupSignPubkey, err = p2pcrypto.MarshalPublicKey(p2ppubkey)
			if err != nil {
				output[ERROR_INFO] = "group key can't be decoded, err:" + err.Error()
				return c.JSON(http.StatusBadRequest, output)
			}
		}

		var buffer bytes.Buffer
		buffer.Write(groupSignPubkey)
		buffer.Write([]byte(params.GroupId))
		hash := chain.Hash(buffer.Bytes())
		signature, err := ks.SignByKeyName(params.GroupId, hash)
		encodedString := hex.EncodeToString(signature)

		return c.JSON(http.StatusOK, &LeaveGroupResult{GroupId: params.GroupId, Signature: encodedString})
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", params.GroupId)
		return c.JSON(http.StatusBadRequest, output)
	}
}
