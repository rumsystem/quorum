package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// NewHTTPClient return *http.Client with `cacert` config
func NewHTTPClient() (*http.Client, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	return client, nil
}

// RequestAPI sends a request to the API
func RequestAPI(url string, method string, payload interface{}, headers http.Header, result interface{}) (int, []byte, error) {
	upperMethod := strings.ToUpper(method)
	methods := map[string]string{
		"HEAD":    http.MethodHead,
		"GET":     http.MethodGet,
		"POST":    http.MethodPost,
		"PUT":     http.MethodPut,
		"DELETE":  http.MethodDelete,
		"PATCH":   http.MethodPatch,
		"OPTIONS": http.MethodOptions,
	}

	if _, found := methods[upperMethod]; !found {
		panic(fmt.Sprintf("not support http method: %s", method))
	}

	logger.Debugf("request %s %s headers: %+v payload: %+v", method, url, headers, payload)

	method = methods[upperMethod]

	var payloadBytes []byte
	if payload != nil {
		if data, ok := payload.([]byte); ok { // use []byte payload
			payloadBytes = data
		} else if data, ok := payload.(string); ok { // string => []byte
			payloadBytes = []byte(data)
		} else { // convert go struct to json
			var err error
			payloadBytes, err = json.Marshal(payload)
			if err != nil {
				return 0, nil, err
			}
		}
	}

	client, err := NewHTTPClient()
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, nil, err
	}
	if headers != nil {
		req.Header = headers
	}
	if headers.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	if resp.StatusCode >= 400 {
		return resp.StatusCode, content, nil
	}

	if result != nil && len(content) > 0 {
		if err := json.Unmarshal(content, result); err != nil {
			return resp.StatusCode, content, err
		}
	}

	logger.Debugf("response status: %d body: %s", resp.StatusCode, content)

	return resp.StatusCode, content, nil
}
