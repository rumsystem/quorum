package api

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"github.com/labstack/echo/v4"
    "github.com/huo-ju/quorum/internal/pkg/data"
)

func (h *Handler) Create(c echo.Context) (err error) {

	var bodyBytes []byte
	if c.Request().Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request().Body)
	}

    c.Echo().Logger.Info(bodyBytes)

    var activity data.Activity
    err = json.Unmarshal(bodyBytes, &activity)
	if err != nil { // error joson data
		return c.JSON(http.StatusOK, map[string]string{"create": fmt.Sprintf("%s",err), })
	}

    err = h.PubsubTopic.Publish(h.Ctx, bodyBytes)
	fmt.Println(err)

    return c.JSON(http.StatusOK, map[string]int64{"post": 0, })
}
