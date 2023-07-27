package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type SeedUrlextendParam struct {
	SeedURL string `json:"seed" validate:"required,url" example:"rum://seed?v=1&e=0&n=0&b=tknSczG2RC6hEBTXZyig7w&c=Za8zI2nAWaTNSvSv6cnPPxHCZef9sGtKtgsZ8iSxj0E&g=SfGcugfLTZ68Hc-xscFwMQ&k=AnRP4sojIvAH-Ugqnd7ZaM1H8j_c1pX6clyeXgAORiGZ&s=mrcA0LDzo54zUujZTINvWM_k2HSifv2T4JfYHAY2EzsCRGdR5vxHbvVNStlJOOBK_ohT6vFGs0FDk2pWYVRPUQE&t=FyvyFrtDGC0&a=timeline.dev&y=group_timeline&u=http%3A%2F%2F1.2.3.4%3A6090%3Fjwt%3DeyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhbGxvd0dyb3VwcyI6WyI0OWYxOWNiYS0wN2NiLTRkOWUtYmMxZC1jZmIxYjFjMTcwMzEiXSwiZXhwIjoxODI3Mzc0MjgyLCJuYW1lIjoiYWxsb3ctNDlmMTljYmEtMDdjYi00ZDllLWJjMWQtY2ZiMWIxYzE3MDMxIiwicm9sZSI6Im5vZGUifQ.rr_tYm0aUdmOeM0EYVzNpKmoNDOpSGzD38s6tjlxuCo"`
}

type SeedUrlextendResult struct {
	Seed         *quorumpb.GroupSeed `json:"seed" validate:"required"`
	ChainapiUrls []string            `json:"urls" validate:"required,gte=1,dive,required,url" example:"http://1.2.3.4:6090?jwt=xxx|https://xxx.com?jwt=yyy"` // multi items separate by `|`
}

func (h *Handler) SeedUrlextend(c echo.Context) error {
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
		Seed:         seed,
		ChainapiUrls: urls,
	}

	return c.JSON(http.StatusOK, res)
}
