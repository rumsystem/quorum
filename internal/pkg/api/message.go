package api

import (
	"net/http"
	"github.com/labstack/echo/v4"
)

func Post(c echo.Context) (err error) {
    //msg := new(briko.Message)
    //if err = c.Bind(msg); err != nil {
    //    return
    //}

    //insertID, err := msg.Save()
    //if err != nil {
    //    c.Echo().Logger.Error("Insert Error", err)
    //}
    //return c.JSON(http.StatusOK, map[string]int64{"result_id": insertID, })
    return c.JSON(http.StatusOK, map[string]int64{"post": 0, })
}
