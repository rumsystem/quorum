package appdata

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	"github.com/huo-ju/quorum/internal/pkg/utils"
	logging "github.com/ipfs/go-log/v2"
)

var appsynclog = logging.Logger("appsync")

type AppSync struct {
	appdb   *AppDb
	apiroot string
}

func NewAppSyncAgent(apiroot string, appdb *AppDb) *AppSync {
	appsync := &AppSync{appdb, apiroot}
	return appsync
}

func (appsync *AppSync) GetGroupsIds() ([]string, error) {
	apiurl := fmt.Sprintf("%s/groups", appsync.apiroot)
	req, err := http.NewRequest("GET", apiurl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	client := utils.NewHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			groupids := []string{}
			var groups map[string][]quorumpb.GroupItem
			err = json.Unmarshal(body, &groups)
			if err != nil {
				return nil, err
			}
			for _, g := range groups {
				for _, gi := range g {
					groupids = append(groupids, gi.GroupId)
				}
			}
			return groupids, err
		} else {
			return nil, err
		}
	} else {
		//404?
		return nil, fmt.Errorf("api response err: %d", resp.StatusCode)
	}
}

func (appsync *AppSync) SyncBlock(groupid string, blocknum uint64) error {
	for {
		apiurl := fmt.Sprintf("%s/block/%s/%d", appsync.apiroot, groupid, blocknum)
		req, err := http.NewRequest("GET", apiurl, nil)
		if err != nil {
			return err
		}
		req.Header.Add("Content-Type", "application/json")
		client := utils.NewHTTPClient()

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.Body != nil {
			defer resp.Body.Close()
		}
		if resp.StatusCode == 200 {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			var b *quorumpb.Block
			err = json.Unmarshal(body, &b)
			if err != nil {
				return err
			}
			//TODO: Verify block?
			for _, trx := range b.Trxs {
				err = appsync.appdb.AddMetaByTrx(blocknum, groupid, trx)
				if err != nil {
					return err
				}
				appsynclog.Infof("add group %s trx %s to the appdb.", groupid, trx.TrxId)
			}
			blocknum += 1
		} else {
			appsynclog.Infof("read group %s block %d err or no new blocks.", groupid, blocknum)
			return nil
		}
	}
}

func (appsync *AppSync) Start(interval int) {
	go func() {
		for {
			groupids, err := appsync.GetGroupsIds()
			if err != nil {
				appsynclog.Errorf("request /groups api err: %s", err)
			} else {
				for _, groupid := range groupids {
					max, err := appsync.appdb.GetMaxBlockNum(groupid)
					err = appsync.SyncBlock(groupid, max+1)
					if err != nil {
						appsynclog.Errorf("sync group: %s number %d err %s", groupid, max+1, err)
					}

				}
			}
			time.Sleep(time.Duration(interval) * time.Second)

		}
	}()
}
