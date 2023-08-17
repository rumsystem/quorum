package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	handlers "github.com/rumsystem/quorum/pkg/chainapi/handlers"
	_ "github.com/rumsystem/quorum/pkg/pb" //import for swaggo
)

// @Tags Keystore
// @Summary CreateSignKey
// @Description Create a new eth sign key pair
// @Accept json
// @Produce json
// @Param data body handlers.CreateSignKeyParams true "CreateSignKeyParams"
// @Success 200 {object} handlers.CreateSignKeyResult
// @Router /api/v2/rumlite/keystore/createsignkey [post]
func (h *Handler) CreateSignKey() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)

		params := new(handlers.CreateSignKeyParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		result, err := handlers.CreateSignKey(params, options.GetNodeOptions())
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, result)
	}
}

// @Tags Keystore
// @Summary GetPubkeyByKeyName
// @Description Get pubkey by given keyname
// @Accept json
// @Produce json
// @Param data body handlers.GetPubkeyByKeyNameParams true "GetPubkeyByKeyNameParams"
// @Success 200 {object} handlers.GetPubkeyByKeyNameResult
// @Router /api/v2/rumlite/keystore/getkeybykeyname [post]
func (h *Handler) GetPubkeyByKeyName() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)
		params := new(handlers.GetPubkeyByKeyNameParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		result, err := handlers.GetPubkeyByKeyName(params, options.GetNodeOptions())
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, result)
	}
}

// @Tags Keystore
// @Summary GetPubkeyByKeyName
// @Description Get all pubkeys
// @Accept json
// @Produce json
// @Param data body handlers.GetAllKeysParams true "GetAllKeysParams"
// @Success 200 {object} handlers.GetAllKeysResult
// @Router /api/v2/rumlite/keystore/getallkeys [post]
func (h *Handler) GetAllKeys() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)
		params := new(handlers.GetAllKeysParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		result, err := handlers.GetAllKeys(params, options.GetNodeOptions())
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, result)
	}
}
