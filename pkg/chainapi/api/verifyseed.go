package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	rumchaindata "github.com/rumsystem/quorum/pkg/data"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type VerifyGroupSeedParam struct {
	Seed []byte `param:"seed" validate:"required" example:"seed"`
}

type VerifyGroupSeedResult struct {
	Verified bool   `json:"verified"`
	Error    string `json:"error"`
}

func (h *Handler) VerifyGroupSeed(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(VerifyGroupSeedParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	result := &VerifyGroupSeedResult{}

	seed := &quorumpb.GroupSeed{}
	err = proto.Unmarshal(params.Seed, seed)
	if err != nil {
		result.Verified = false
		result.Error = err.Error()
		return cc.JSON(http.StatusOK, result)
	}

	verified, err := rumchaindata.VerifyGroupSeed(seed)
	if err != nil {
		result.Verified = false
		result.Error = err.Error()
		return cc.JSON(http.StatusOK, result)
	}

	result.Verified = verified
	result.Error = ""

	return cc.JSON(http.StatusOK, result)
}
