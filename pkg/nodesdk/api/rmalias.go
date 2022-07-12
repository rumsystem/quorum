package nodesdkapi

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type RmAliasParams struct {
	Alias string `json:"alias" validate:"required"`
}

func (h *NodeSDKHandler) RmAlias() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)
		params := new(RmAliasParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		ks := nodesdkctx.GetKeyStore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			return rumerrors.NewBadRequestError("Open keystore failed")
		}

		password := os.Getenv("RUM_KSPASSWD")

		if err := dirks.UnAlias(params.Alias, password); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		return c.JSON(http.StatusOK, "done")
	}
}
