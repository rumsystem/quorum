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

	validate := validator.New()
	params := new(LeaveGroupParam)

	if err := c.Bind(params); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	if err = validate.Struct(params); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	groupmgr := chain.GetGroupMgr()
	group, ok := groupmgr.Groups[params.GroupId]
	if !ok {
		output := ErrorResponse{Error: fmt.Sprintf("Group %s not exist", params.GroupId)}
		return c.JSON(http.StatusBadRequest, output)
	}

	if err := group.LeaveGrp(); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	delete(groupmgr.Groups, params.GroupId)

	var groupSignPubkey []byte
	ks := localcrypto.GetKeystore()
	dirks, ok := ks.(*localcrypto.DirKeyStore)
	if !ok {
		output := ErrorResponse{Error: fmt.Sprintf("unknown keystore type  %v:", ks)}
		return c.JSON(http.StatusBadRequest, output)
	}

	hexkey, err := dirks.GetEncodedPubkey("default", localcrypto.Sign)
	pubkeybytes, err := hex.DecodeString(hexkey)
	p2ppubkey, err := p2pcrypto.UnmarshalSecp256k1PublicKey(pubkeybytes)
	groupSignPubkey, err = p2pcrypto.MarshalPublicKey(p2ppubkey)
	if err != nil {
		output := ErrorResponse{Error: "group key can't be decoded, err:" + err.Error()}
		return c.JSON(http.StatusBadRequest, output)
	}

	var buffer bytes.Buffer
	buffer.Write(groupSignPubkey)
	buffer.Write([]byte(params.GroupId))
	hash := chain.Hash(buffer.Bytes())
	signature, err := ks.SignByKeyName(params.GroupId, hash)
	encodedString := hex.EncodeToString(signature)

	// delete group seed from appdata
	if err := h.Appdb.DelGroupSeed(params.GroupId); err != nil {
		output := ErrorResponse{Error: fmt.Sprintf("save group seed failed: %s", err)}
		return c.JSON(http.StatusBadRequest, output)
	}

	return c.JSON(http.StatusOK, &LeaveGroupResult{GroupId: params.GroupId, Signature: encodedString})
}
