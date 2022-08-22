package nodesdkapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
)

type SeedUrlextendParam struct {
	SeedURL string `json:"seed" validate:"required,url"`
}

type SeedUrlextendResult struct {
	Seed         handlers.GroupSeed `json:"seed" validate:"required"`
	ChainapiUrls []string           `json:"urls" validate:"required,gte=1,dive,required,url"`
}

func (h *NodeSDKHandler) SeedUrlextend(c echo.Context) error {
	cc := c.(*utils.CustomContext)

	param := new(SeedUrlextendParam)
	if err := cc.BindAndValidate(param); err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	seed, urls, err := handlers.UrlToGroupSeed(param.SeedURL)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	res := SeedUrlextendResult{
		Seed:         *seed,
		ChainapiUrls: urls,
	}

	return c.JSON(http.StatusOK, res)
}
