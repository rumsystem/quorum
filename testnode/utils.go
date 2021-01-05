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

	"github.com/huo-ju/quorum/internal/pkg/utils"
)

func RequestAPI(apiurl string, endpoint string, method string, data string) ([]byte, error) {
	url := fmt.Sprintf("%s%s", apiurl, endpoint)
	switch method {
	case "GET":
		log.Printf("%s %s", method, url)

		req, err := http.NewRequest("GET", url, bytes.NewBufferString(data))
		if err != nil {
			return []byte(""), err
		}
		req.Header.Add("Content-Type", "application/json")
		client, err := utils.NewHTTPClient()
		if err != nil {
			return []byte(""), err
		}

		resp, err := client.Do(req)
		if err != nil {
			return []byte(""), err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return []byte(""), err
		}
		return body, nil
	case "POST":
		log.Printf("%s %s", method, url)
		client, err := utils.NewHTTPClient()
		if err != nil {
			return []byte(""), err
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(data))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return []byte(""), err
		}
		return body, nil
	}
	return []byte(""), nil
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
