package utils

import (
	"net/http"
	"time"
)

// NewHTTPClient return *http.Client with `cacert` config
func NewHTTPClient() (*http.Client, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	return client, nil
}
