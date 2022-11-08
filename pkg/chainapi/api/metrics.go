package api

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/prom2json"
)

func (h *Handler) Metrics(c echo.Context) error {
	out := &bytes.Buffer{}
	metricFamilies, _ := prometheus.DefaultGatherer.Gather()
	for i := range metricFamilies {
		expfmt.MetricFamilyToText(out, metricFamilies[i])

	}

	contentType := strings.ToLower(c.Request().Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		mfChan := make(chan *dto.MetricFamily, 1024)
		if err := prom2json.ParseReader(out, mfChan); err != nil {
			return err
		}
		result := []*prom2json.Family{}
		for mf := range mfChan {
			result = append(result, prom2json.NewFamily(mf))
		}
		return c.JSON(http.StatusOK, result)
	} else { // plain text
		return c.String(http.StatusOK, string(out.Bytes()))
	}
}
