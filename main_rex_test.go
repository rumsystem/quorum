package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	api "github.com/rumsystem/quorum/pkg/chainapi/api"
	"github.com/rumsystem/quorum/testnode"
)

var (
	pidlist                                   []int
	bootstrapapi, peer1api, peer2api          string
	peerapilist, groupIds                     []string
	timerange, nodes, groups, posts, synctime int
)

func TestMain(m *testing.M) {
	timerangePtr := flag.Int("timerange", 5, "interval(in normal distribution) of sending transactions")
	nodesPtr := flag.Int("nodes", 2, "mock nodes")
	groupsPtr := flag.Int("groups", 5, "groups on each node")
	postsPtr := flag.Int("posts", 5, "posts on each group")
	synctimePtr := flag.Int("synctime", 30, "time to wait before verify")
	rextestmode := flag.Bool("rextest", false, "RumExchange Test Mode")

	flag.Parse()

	timerange = *timerangePtr
	nodes = *nodesPtr
	groups = *groupsPtr
	posts = *postsPtr
	synctime = *synctimePtr

	log.Printf("Setup testing nodes: %d, groups: %d, posts: %d\n", nodes, groups, posts)
	log.Println(pidlist)
	pidch := make(chan int)
	go func() {
		for {
			select {
			case pid := <-pidch:
				log.Println("receive pid", pid)
				pidlist = append(pidlist, pid)
				if len(pidlist) == 3 {
					return
				}
			}
		}
	}()

	cliargs := testnode.Nodecliargs{Rextest: *rextestmode}
	var tempdatadir string
	bootstrapapi, peerapilist, tempdatadir, _ = testnode.RunNodesWithBootstrap(context.Background(), cliargs, pidch, nodes)
	log.Println("peers: ", peerapilist)
	exitVal := m.Run()
	log.Println("after tests clean:", tempdatadir)
	testnode.Cleanup(tempdatadir, peerapilist)
	os.Exit(exitVal)
}

// create n groups on each peer, post contents, then join all groups, wait for sync, and verify peerN groups == peerM groups
func TestGroupsContentsRexSync(t *testing.T) {

	log.Printf("_____________TestGroupsContentsRexSync_RUNNING_____________")

	var seedsByNode [][]string

	groupspeernum := groups

	for idx, peerapi := range peerapilist {
		var seeds []string
		for i := 0; i < groupspeernum; i++ {

			groupName := fmt.Sprintf("testgroup_peer_%d_%d", idx+1, i+1)
			_, resp, err := testnode.RequestAPI(peerapi, "/api/v1/group", "POST", fmt.Sprintf(`{"group_name":"%s","app_key":"default", "consensus_type":"poa","encryption_type":"public"}`, groupName))
			if err == nil {
				var objmap map[string]interface{}
				if err := json.Unmarshal(resp, &objmap); err != nil {
					t.Errorf("Data Unmarshal error %s", err)
				} else {
					seeds = append(seeds, string(resp))
					seedurl := objmap["seed"]
					groupId := testnode.SeedUrlToGroupId(seedurl.(string))
					groupIds = append(groupIds, groupId)
					log.Printf("group %s(%s) created on peer%d", groupName, groupId, idx+1)
				}
			} else {
				t.Errorf("create group on peer%d error %s", 1, err)
			}
		}
		seedsByNode = append(seedsByNode, seeds)
		time.Sleep(1 * time.Second)
	}

	log.Printf("_____________create group done_____________")
	ready := "IDLE"
	waitingcounter := 10

	ok := true
	for {
		if waitingcounter <= 0 {
			if ok == false {
				t.Errorf("some groups status is not IDLE.")
			}
			break
		}

		for _, peerapi := range peerapilist {
			_, resp, err := testnode.RequestAPI(peerapi, "/api/v1/groups", "GET", "")

			if err != nil {
				t.Errorf("get peer group error %s", err)
			}

			groupslist := &api.GroupInfoList{}
			if err := json.Unmarshal(resp, &groupslist); err != nil {
				if len(groupslist.GroupInfos) != groupspeernum {
					t.Errorf("Group number check failed, have %d groups, except %d ", len(groupslist.GroupInfos), groupspeernum)
				}

				for _, groupinfo := range groupslist.GroupInfos {
					log.Printf("Group %s status %s", groupinfo.GroupId, groupinfo.GroupStatus)
					if groupinfo.GroupStatus != ready {
						log.Printf("group %s status is %s not ready.", groupinfo.GroupId, groupinfo.GroupStatus)
						ok = false
					}
				}
				t.Errorf("parse peer group error %s", err)
			}
		}

		if ok {
			break
		} else {
			t.Logf("wait 3s for groups IDLE")
			time.Sleep(3 * time.Second)
		}
		waitingcounter -= 1
	}
	log.Printf("_____________group status verify done_____________")

	groupIdToTrxIds := map[string][]string{}
	for peeridx, peerapi := range peerapilist {
		_, resp, err := testnode.RequestAPI(peerapi, "/api/v1/groups", "GET", "")

		if err != nil {
			t.Errorf("get peer group error %s", err)
		}

		groupslist := &api.GroupInfoList{}
		err = json.Unmarshal(resp, &groupslist)

		if err == nil {
			for _, groupinfo := range groupslist.GroupInfos {
				groupIdToTrxIds[groupinfo.GroupId] = []string{}
				for i := 1; i <= posts; i++ {
					content := fmt.Sprintf(`{"type":"Add","object":{"type":"Note","content":"peer%d_content_%s_%d","name":"peer%d_name_%s_%d"},"target":{"id":"%s","type":"Group"}}`, peeridx, groupinfo.GroupId, i, peeridx, groupinfo.GroupId, i, groupinfo.GroupId)
					_, resp, err := testnode.RequestAPI(peerapi, "/api/v1/group/content", "POST", content)
					if err != nil {
						t.Errorf("post content to api error %s", err)
					}
					var objmap map[string]interface{}
					if err = json.Unmarshal(resp, &objmap); err != nil {
						// store trx id, verify it later on each group
						t.Errorf("Data Unmarshal error %s", err)
					}
					if objmap["trx_id"] != nil {
						t.Logf("post with trxid: %s created", objmap["trx_id"].(string))
						groupIdToTrxIds[groupinfo.GroupId] = append(groupIdToTrxIds[groupinfo.GroupId], objmap["trx_id"].(string))
					} else {
						t.Errorf("Resp body was not included trx_id %s", string(resp))
					}
					// use normal distribution time range
					// half range  == 3 * stddev (99.7%)
					mean := float64(timerange) / 2.0
					stddev := mean / 3.0
					sleepTime := rand.NormFloat64()*stddev + mean + 5
					log.Printf("sleep: %.2f s before next post\n", sleepTime)
					time.Sleep(time.Duration(sleepTime*1000) * time.Millisecond)
				}
			}
		}
	}

	log.Printf("Wait 20s for sync")
	time.Sleep(20 * time.Second)

	grouptrxsbefore := make(map[string]int)
	for peeridx, peerapi := range peerapilist {
		_, resp, err := testnode.RequestAPI(peerapi, "/api/v1/groups", "GET", "")
		_ = peeridx
		if err != nil {
			t.Errorf("get peer group error %s", err)
		}
		groupslist := &api.GroupInfoList{}
		err = json.Unmarshal(resp, &groupslist)
		if err == nil {
			for _, groupinfo := range groupslist.GroupInfos {
				trxids := testnode.GetAllGroupTrxIds(context.Background(), peerapi, groupinfo.GroupId, groupinfo.HighestBlockId)
				grouptrxsbefore[groupinfo.GroupId] = len(*trxids)
				//Trxs
			}
		}
	}

	log.Printf("_____________join all groups_____________")
	for peerIdx, peerapi := range peerapilist {
		for seedIdx := 0; seedIdx < nodes; seedIdx++ {
			if seedIdx != peerIdx {
				seedsFromOtherNode := seedsByNode[seedIdx]
				if len(seedsFromOtherNode) >= groupspeernum {
					for i := 0; i < groupspeernum; i++ {
						g := seedsFromOtherNode[i]
						// join to other groups of other nodes
						_, _, err := testnode.RequestAPI(peerapi, "/api/v2/group/join", "POST", g)
						if err != nil {
							t.Errorf("peer%d join group %s error %s", peerIdx+1, g, err)
						} else {
							t.Logf("peer%d join group %s", peerIdx+1, g)
						}
					}
				}
			}
		}
	}

	log.Printf("Wait 20s for sync")
	time.Sleep(20 * time.Second)

	grouptrxsafter := make(map[string]int)
	for peeridx, peerapi := range peerapilist {
		_, resp, err := testnode.RequestAPI(peerapi, "/api/v1/groups", "GET", "")
		_ = peeridx
		if err != nil {
			t.Errorf("get peer group error %s", err)
		}
		groupslist := &api.GroupInfoList{}
		err = json.Unmarshal(resp, &groupslist)
		if err == nil {
			for _, groupinfo := range groupslist.GroupInfos {
				trxids := testnode.GetAllGroupTrxIds(context.Background(), peerapi, groupinfo.GroupId, groupinfo.HighestBlockId)
				grouptrxsafter[fmt.Sprintf("%s_%s", groupinfo.GroupId, peerapi)] = len(*trxids)
				//Trxs
			}
		}
	}

	for key, v := range grouptrxsafter {
		keys := strings.Split(key, "_")
		groupid := keys[0]
		beforenum := grouptrxsbefore[groupid]
		if v != beforenum {
			t.Errorf("not match key %s value %d expect %d", key, v, beforenum)
		}
		log.Printf("peer %s group %s trxs number: %d", keys[1], keys[0], v)
	}
}
