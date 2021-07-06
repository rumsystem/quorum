package testnode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
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
		client := &http.Client{}

		//resp, err := http.Get(url)
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
		resp, err := http.Post(url, "application/json", bytes.NewBufferString(data))

		if err != nil {
			return []byte(""), err
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

func CheckNodeRunning(ctx context.Context, url string) bool {
	apiurl := fmt.Sprintf("%s/api/v1", url)
	fmt.Printf("checkNodeRunning: %s\n", apiurl)
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return false
		case <-ticker.C:
			resp, err := RequestAPI(apiurl, "/node", "GET", "")
			if err == nil {
				var objmap map[string]interface{}
				if err := json.Unmarshal(resp, &objmap); err != nil {
					fmt.Println(err)
				} else {
					log.Println(objmap)
					if objmap["node_status"] == "NODE_ONLINE" && objmap["node_type"] == "bootstrap" {
						ticker.Stop()
						return true
					} else if objmap["peers"] != nil {
						peers := []string{}
						for _, peer := range objmap["peers"].([]interface{}) {
							peers = append(peers, peer.(string))
						}
						if len(peers) >= 0 {
							ticker.Stop()
							return true
						}
					}
				}
			}
		}
	}
}
