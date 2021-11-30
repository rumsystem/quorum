package testnode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/utils"
)

func RequestAPI(apiurl string, endpoint string, method string, data string) ([]byte, error) {
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

	method = methods[upperMethod]

	url := fmt.Sprintf("%s%s", apiurl, endpoint)
	if len(data) > 0 {
		log.Printf("request %s %s with body: %s", method, url, data)
	} else {
		log.Printf("request %s %s", method, url)

	}
	client, err := utils.NewHTTPClient()
	if err != nil {
		return []byte(""), err
	}

	req, err := http.NewRequest(method, url, bytes.NewBufferString(data))
	if err != nil {
		return []byte(""), err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(req)
	if err != nil {
		return []byte(""), err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte(""), err
	}
	log.Printf("response status: %d body: %s", resp.StatusCode, body)
	return body, nil
}

func CheckNodeRunning(ctx context.Context, url string) (string, bool) {
	apiurl := fmt.Sprintf("%s/api/v1", url)
	fmt.Printf("checkNodeRunning: %s\n", apiurl)
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return "", false
		case <-ticker.C:
			resp, err := RequestAPI(apiurl, "/node", "GET", "")
			if err == nil {
				var objmap map[string]interface{}
				if err := json.Unmarshal(resp, &objmap); err != nil {
					fmt.Println(err)
				} else {
					if objmap["node_status"] == "NODE_ONLINE" && objmap["node_type"] == "bootstrap" {
						ticker.Stop()
						return objmap["node_id"].(string), true
					} else if objmap["peers"] != nil {
						for key, peers := range objmap["peers"].(map[string]interface{}) {
							reqpeers := []string{}
							if strings.Index(key, "/quorum/meshsub/") >= 0 {
								for _, p := range peers.([]interface{}) {
									reqpeers = append(reqpeers, p.(string))
								}
							}
							if len(reqpeers) >= 0 {
								ticker.Stop()
								return objmap["node_id"].(string), true
							}
						}
					}
				}
			}
		}
	}
}
