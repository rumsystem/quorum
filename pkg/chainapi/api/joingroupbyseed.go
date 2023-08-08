package api

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type JoinGroupBySeedParam struct {
	Seed []byte `from:"seed" json:"seed" validate:"required"`
}
type JoinGroupBySeedResult struct {
	GroupItem *quorumpb.GroupItemRumLite `json:"groupItem"`
}

// @Tags Groups
// @Summary JoinGroupBySeed
// @Description Join a group by using group seed
// @Accept json
// @Produce json
// @Param data body handlers.JoinGroupBySeedParam true "JoinGroupBySeedParam"
// @Success 200 {object} JoinGroupBySeedResult
// @Router /api/v2/group/join [post]
func (h *Handler) JoinGroupBySeed() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		payload := new(JoinGroupBySeedParam)
		if err := cc.BindAndValidate(payload); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		//unmarshal seed
		seed := &quorumpb.GroupSeedRumLite{}
		err := proto.Unmarshal(payload.Seed, seed)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		groupItem := seed.Group

		//check if group exist
		groupmgr := chain.GetGroupMgr()
		if _, ok := groupmgr.Groups[groupItem.GroupId]; ok {
			msg := fmt.Sprintf("group with group_id <%s> already exist", groupItem.GroupId)
			return rumerrors.NewBadRequestError(msg)
		}

		//verify hash and signature
		hash := localcrypto.Hash(payload.Seed)
		if !bytes.Equal(hash, seed.Hash) {
			msg := fmt.Sprintf("hash not match, expect %s, got %s", hex.EncodeToString(hash), hex.EncodeToString(seed.Hash))
			return rumerrors.NewBadRequestError(msg)
		}

		verified, err := rumchaindata.VerifySign(groupItem.OwnerPubKey, seed.Signature, hash)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if !verified {
			return rumerrors.NewBadRequestError("verify signature failed")
		}

		//verify genesis block
		r, err := rumchaindata.ValidGenesisBlockRumLite(groupItem.GenesisBlock)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if !r {
			msg := "Join Group failed, verify genesis block failed"
			return rumerrors.NewBadRequestError(msg)
		}

		//create empty group
		group := &chain.GroupRumLite{}
		err = group.JoinGroup(groupItem)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		return cc.JSON(http.StatusOK, JoinGroupBySeedResult{GroupItem: groupItem})

	}
}
