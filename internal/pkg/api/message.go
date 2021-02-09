package api

import (
	"encoding/json"
	"fmt"
	"github.com/huo-ju/quorum/internal/pkg/data"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"net/http"
)

func (h *Handler) Create(c echo.Context) (err error) {

	var bodyBytes []byte
	if c.Request().Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request().Body)
	}

	//c.Echo().Logger.Info(bodyBytes)

	var activity data.Activity
	err = json.Unmarshal(bodyBytes, &activity)
	if err != nil { // error joson data
		return c.JSON(http.StatusOK, map[string]string{"create": fmt.Sprintf("%s", err)})
	}

	h.PubsubTopic.Publish(h.Ctx, bodyBytes)
	return c.JSON(http.StatusOK, map[string]int64{"post": 0})
}
