package testnode

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	guuid "github.com/google/uuid"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

func RequestAPI(baseUrl string, endpoint string, method string, data string) (int, []byte, error) {
	_url := fmt.Sprintf("%s%s", baseUrl, endpoint)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json; charset=utf-8")
	statusCode, content, err := utils.RequestAPI(_url, method, []byte(data), headers, nil)
	if err != nil {
		return 0, []byte(""), err
	}

	log.Printf("response status: %d body: %s", statusCode, content)
	return statusCode, content, nil
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
			_, resp, err := RequestAPI(apiurl, "/node", "GET", "")
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

func CheckApiServerRunning(ctx context.Context, baseUrl string) bool {
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return false
		case <-ticker.C:
			statusCode, resp, err := RequestAPI(baseUrl, "/api/v1/node", "GET", "")
			if err != nil {
				fmt.Println(err)
				continue
			}
			if statusCode == 200 {
				var objmap map[string]interface{}
				if err := json.Unmarshal(resp, &objmap); err != nil {
					fmt.Println(err)
				}

				ticker.Stop()
				return true
			}
		}
	}
}

func GetAllGroupTrxIds(ctx context.Context, baseUrl string, group_id string, height_blockid string) *[]string {
	trxids := []string{}
	_, resp, err := RequestAPI(baseUrl, fmt.Sprintf("/api/v1/block/%s/%s", group_id, height_blockid), "GET", "")
	if err != nil {
		return &trxids
	}
	block := &quorumpb.Block{}
	if err := json.Unmarshal(resp, &block); err == nil {
		prevBlockId := block.PrevBlockId
		for {
			_, resp, err := RequestAPI(baseUrl, fmt.Sprintf("/api/v1/block/%s/%s", group_id, prevBlockId), "GET", "")
			if err != nil {
				break
			}
			err = json.Unmarshal(resp, &block)
			if err != nil {
				break
			}
			if prevBlockId == "" || prevBlockId == block.PrevBlockId {
				break
			}

			for _, trx := range block.Trxs {
				trxids = append(trxids, trx.TrxId)
			}
			prevBlockId = block.PrevBlockId
		}

	}

	return &trxids
}

func SeedUrlToGroupId(seedurl string) string {
	if !strings.HasPrefix(seedurl, "rum://seed?") {
		return ""
	}
	u, err := url.Parse(seedurl)
	if err != nil {
		return ""
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ""
	}
	b64gstr := q.Get("g")

	b64gbyte, err := base64.RawURLEncoding.DecodeString(b64gstr)
	b64guuid, err := guuid.FromBytes(b64gbyte)
	return b64guuid.String()
}
