package nodesdkapi

import (
	"os"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type BindKeyAliasParams struct {
	Alias   string `json:"alias" validate:"required"`
	KeyName string `json:"keyname" validate:"required"`
	Type    string `json:"type"  validate:"required,oneof=encrypt sign"`
}

func (h *NodeSDKHandler) BindAliasWithKeyName() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)
		params := new(BindKeyAliasParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		ks := nodesdkctx.GetKeyStore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			return rumerrors.NewBadRequestError("Open keystore failed")
		}

		password := os.Getenv("RUM_KSPASSWD")

		keyname := dirks.AliasToKeyname(params.Alias)
		if keyname != "" {
			if err := dirks.UnAlias(params.Alias, password); err != nil {
				return rumerrors.NewBadRequestError(err)
			}
		}

		if err := dirks.NewAlias(params.Alias, params.KeyName, password); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		return cc.Success()
	}
}
