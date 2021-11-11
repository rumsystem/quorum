package api

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	chain "github.com/rumsystem/quorum/internal/pkg/chain"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
)

type ClearGroupDataParam struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
}

type ClearGroupDataResult struct {
	GroupId   string `json:"group_id" validate:"required"`
	Signature string `json:"signature" validate:"required"`
}

// @Tags Groups
// @Summary ClearGroupData
// @Description Clear group data
// @Produce json
// @Success 200 {object} ClearGroupDataResult
// @Router /v1/group/clear [post]
func (h *Handler) ClearGroupData(c echo.Context) (err error) {
	output := make(map[string]string)
	validate := validator.New()
	params := new(ClearGroupDataParam)

	if err := c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if !ok {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", params.GroupId)
		return c.JSON(http.StatusBadRequest, output)
	}

	if err := group.ClearGroup(); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	var groupSignPubkey []byte
	ks := localcrypto.GetKeystore()
	dirks, ok := ks.(*localcrypto.DirKeyStore)
	if ok {
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

	return c.JSON(http.StatusOK, &ClearGroupDataResult{GroupId: params.GroupId, Signature: encodedString})
}
