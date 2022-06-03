package nodesdkapi

import (
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type RmAliasParams struct {
	Alias string `json:"alias" validate:"required"`
}

func (h *NodeSDKHandler) RmAlias() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)

		validate := validator.New()
		params := new(RmAliasParams)

		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		ks := nodesdkctx.GetKeyStore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			output[ERROR_INFO] = "Open keystore failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		password := os.Getenv("RUM_KSPASSWD")

		err = dirks.UnAlias(params.Alias, password)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		return c.JSON(http.StatusOK, "done")
	}
}
