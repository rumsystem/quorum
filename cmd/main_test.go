package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	api "github.com/rumsystem/quorum/internal/pkg/api"
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

/*
	To Verify :
		Node create a group and leave it, should be able to rejoin and group
*/

type RespError struct {
	error string `json:"error"`
}

func TestJoinGroup(t *testing.T) {
	log.Printf("_____________TestJoinGroup_RUNNING_____________")

	//initial
	groupToCreate := 1

	//create 1 group on each peer, join the group then leave, repeat 3 times and verify the group exist and in "IDLE" status
	for idx, peerapi := range peerapilist {
		for i := 0; i < groupToCreate; i++ {
			log.Printf("_____________CREATE_GROUP_____________")
			var groupseed string
			var groupId string
			status, resp, err := testnode.RequestAPI(peerapi, "/api/v1/group", "POST", fmt.Sprintf(`{"group_name":"testgroup_peer_%d_%d","app_key":"default", "consensus_type":"poa","encryption_type":"public"}`, idx+1, i+1))
			if err == nil || status != 200 {
				var objmap map[string]interface{}
				if err := json.Unmarshal(resp, &objmap); err != nil {
					t.Errorf("Data Unmarshal error %s", err)
				} else {
					groupseed = string(resp)
					groupName := objmap["group_name"]
					groupId = objmap["group_id"].(string)
					log.Printf("group %s(%s) created on peer%d", groupName, groupId, idx+1)
				}
			} else {
				t.Errorf("create group on peer%d error %s", 1, err)
			}
			time.Sleep(1 * time.Second)
			// try join the same group just created
			log.Printf("_____________TEST_JOIN_EXIST_GROUP_____________")
			status, resp, err = testnode.RequestAPI(peerapi, "/api/v1/group/join", "POST", groupseed)

			//check if failed
			if status != 400 {
				t.Errorf("Join existed group test failed with err %s", err.Error())
			}
			time.Sleep(1 * time.Second)

			log.Printf("_____________TEST_LEAVE_GROUP_____________")
			status, resp, err = testnode.RequestAPI(peerapi, "/api/v1/group/leave", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
			if status != 200 {
				t.Errorf("Leave group test failed with response code %d", status)
			}
			time.Sleep(1 * time.Second)

			log.Printf("_____________TEST_JOIN_LEAVED_GROUP_____________")
			status, resp, err = testnode.RequestAPI(peerapi, "/api/v1/group/join", "POST", groupseed)
			if status != 200 {
				t.Errorf("join leaved group test failed with response code %d", status)
			}
			// verify group status
			ready := "IDLE"
			_, resp, err = testnode.RequestAPI(peerapi, "/api/v1/groups", "GET", "")

			if err != nil {
				t.Errorf("get peer group error %s", err)
			}

			groupslist := &api.GroupInfoList{}
			if err := json.Unmarshal(resp, &groupslist); err != nil {
				t.Errorf("parse peer group error %s", err)
			}

			//check group number,
			if len(groupslist.GroupInfos) != 1 {
				t.Errorf("Group number check failed, have %d groups, except 1", len(groupslist.GroupInfos))
			}

			for _, groupinfo := range groupslist.GroupInfos {
				log.Printf("Group %s status %s", groupinfo.GroupId, groupinfo.GroupStatus)
				if groupinfo.GroupId != groupId {
					t.Errorf("Check group status failed %s, groupId mismatch", err)
				}
				if groupinfo.GroupStatus != ready {
					t.Errorf("Check group status failed %s, group not IDLE", err)
				}
			}

			log.Printf("_____________TEST_LEAVE_GROUP_____________")
			//leave group
			status, resp, err = testnode.RequestAPI(peerapi, "/api/v1/group/leave", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
			if status != 200 {
				if err != nil {
					t.Errorf("Leave group test failed with response code %d, resp <%s>, err <%s>", status, string(resp), err.Error())
				} else {
					t.Errorf("leave group test failed with response code %d, resp <%s>", status, string(resp))
				}
			}

			log.Printf("_____________TEST_CLEAR_GROUP_____________")
			//clear group data
			status, resp, err = testnode.RequestAPI(peerapi, "/api/v1/group/clear", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
			if status != 200 {
				if err != nil {
					t.Errorf("clean group test failed with response code %d, resp <%s>, err <%s>", status, string(resp), err.Error())
				} else {
					t.Errorf("clean group test failed with response code %d, resp <%s>", status, string(resp))
				}
			}

		}

		time.Sleep(1 * time.Second)
	}
}

// create n groups on each peer, and join all groups, then verify peerN groups == peerM groups
func TestGroupsPostContents(t *testing.T) {

	log.Printf("_____________TestGroupContents_RUNNING_____________")

	var seedsByNode [][]string

	groupspeernum := groups

	for idx, peerapi := range peerapilist {
		var seeds []string
		for i := 0; i < groupspeernum; i++ {
			_, resp, err := testnode.RequestAPI(peerapi, "/api/v1/group", "POST", fmt.Sprintf(`{"group_name":"testgroup_peer_%d_%d","app_key":"default", "consensus_type":"poa","encryption_type":"public"}`, idx+1, i+1))
			if err == nil {
				var objmap map[string]interface{}
				if err := json.Unmarshal(resp, &objmap); err != nil {
					t.Errorf("Data Unmarshal error %s", err)
				} else {
					seeds = append(seeds, string(resp))
					groupName := objmap["group_name"]
					groupId := objmap["group_id"].(string)
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

	for idx, peergenesisblocks := range seedsByNode {
		if len(peergenesisblocks) != groupspeernum {
			t.Fail()
		}
		t.Logf("Expected %d genesisblocks on peer%d got %d", groupspeernum, idx+1, len(peergenesisblocks))
	}

	for peerIdx, peerapi := range peerapilist {
		for seedIdx := 0; seedIdx < nodes; seedIdx++ {
			if seedIdx != peerIdx {
				seedsFromOtherNode := seedsByNode[seedIdx]
				if len(seedsFromOtherNode) >= groupspeernum {
					for i := 0; i < groupspeernum; i++ {
						g := seedsFromOtherNode[i]
						// join to other groups of other nodes
						_, _, err := testnode.RequestAPI(peerapi, "/api/v1/group/join", "POST", g)
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

	ready := "IDLE"

	for i := 0; i < nodes; i++ {
		// wait for all nodes, all groups ready
		// reinit groupStatus here, to check each node
		groupStatus := map[string]bool{} // add ready groups
		for _, groupId := range groupIds {
			groupStatus[groupId] = false
		}
		waitingcounter := 10
		for {
			if waitingcounter <= 0 {
				break
			}
			peerapi := peerapilist[i]
			groupslist := &api.GroupInfoList{}
			_, resp, err := testnode.RequestAPI(peerapi, "/api/v1/groups", "GET", "")
			if err != nil {
				t.Errorf("get peer group error %s", err)
			}
			if err := json.Unmarshal(resp, &groupslist); err != nil {
				t.Errorf("parse peer group error %s", err)
			}
			for _, groupinfo := range groupslist.GroupInfos {
				if _, found := groupStatus[groupinfo.GroupId]; found {
					if groupinfo.GroupStatus == ready {
						groupStatus[groupinfo.GroupId] = true
					}
					t.Logf("group(node%d): %s %s", i+1, groupinfo.GroupName, groupinfo.GroupStatus)
				} else {
					t.Logf("[cache??] group(node%d): %s %s", i+1, groupinfo.GroupName, groupinfo.GroupStatus)
				}
			}
			ok := true
			for k, v := range groupStatus {
				if v == false {
					ok = false
					t.Logf("group id %s not ready on node%d", k, i+1)
				}
			}
			if ok {
				break
			} else {
				t.Logf("wait 3s for sync")
				time.Sleep(3 * time.Second)
			}
			waitingcounter -= 1
		}
	}

	if len(peerapilist) == 0 {
		return
	}
	peer1api := peerapilist[0]

	// create m posts on each group, then verify each group has the same posts
	groupIdToTrxIds := map[string][]string{}
	for _, groupId := range groupIds {
		groupIdToTrxIds[groupId] = []string{}
		for i := 1; i <= posts; i++ {
			content := fmt.Sprintf(`{"type":"Add","object":{"type":"Note","content":"peer1_content_%s_%d","name":"peer1_name_%s_%d"},"target":{"id":"%s","type":"Group"}}`, groupId, i, groupId, i, groupId)
			_, resp, err := testnode.RequestAPI(peer1api, "/api/v1/group/content", "POST", content)
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
				groupIdToTrxIds[groupId] = append(groupIdToTrxIds[groupId], objmap["trx_id"].(string))
			} else {
				t.Errorf("Resp body was not included trx_id %s", string(resp))
			}
			// use normal distribution time range
			// half range  == 3 * stddev (99.7%)
			mean := float64(timerange) / 2.0
			stddev := mean / 3.0
			sleepTime := rand.NormFloat64()*stddev + mean
			log.Printf("sleep: %.2f s before next post\n", sleepTime)
			time.Sleep(time.Duration(sleepTime*1000) * time.Millisecond)
			//time.Sleep(time.Duration(5*1000) * time.Millisecond)
		}
	}
	t.Logf("waiting %d seconds for peers data sync", synctime)
	time.Sleep(time.Duration(synctime) * time.Second)
	log.Println("start verify groups content")

	for _, groupId := range groupIds {
		trxIds := groupIdToTrxIds[groupId]
		// for each node, verify groups content
		for nodeIdx, peerapi := range peerapilist {
			trxStatus := map[string]bool{}
			for _, trxId := range trxIds {
				trxStatus[trxId] = false
				_, resp, err := testnode.RequestAPI(peerapi, fmt.Sprintf("/api/v1/trx/%s/%s", groupId, trxId), "GET", "")
				if err == nil {
					var data map[string]interface{}
					if err := json.Unmarshal(resp, &data); err != nil {
						t.Errorf("Data Unmarshal error %s", err)
					}
					if data["TrxId"] == trxId {
						trxStatus[trxId] = true
					}
				} else {
					t.Errorf("get /api/v1/trx/%s err: %s", trxId, err)
				}
			}

			//t.Logf("start verify node%d, group id: %s", nodeIdx+1, groupId)
			//resp, err := testnode.RequestAPI(peerapi, fmt.Sprintf("/api/v1/group/%s/content", groupId), "GET", "")
			//groupcontentlist := []api.GroupContentObjectItem{}

			//if err == nil {
			//	if err := json.Unmarshal(resp, &groupcontentlist); err != nil {
			//		t.Errorf("Data Unmarshal error %s", err)
			//	}
			//} else {
			//	t.Errorf("get /api/v1/group/content err: %s", err)
			//}
			//for _, contentitem := range groupcontentlist {
			//	if contentitem.Content != nil {
			//		if _, found := trxStatus[contentitem.TrxId]; found {
			//			trxStatus[contentitem.TrxId] = true
			//			t.Logf("trx %s ok", contentitem.TrxId)
			//		} else {
			//			t.Errorf("trx %s not exists in this groups", contentitem.TrxId)
			//		}
			//	}
			//}

			// check trxStatus, if it has some false value
			for k, v := range trxStatus {
				if v == false {
					t.Logf("trx id %s not found on node%d", k, nodeIdx+1)
					//t.Logf("pause for human verify")
					//time.Sleep(10000000 * time.Second)
					t.Fail()
				}
			}

			//Added by cuicat
			//leave group
			status, resp, err := testnode.RequestAPI(peerapi, "/api/v1/group/leave", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
			if status != 200 {
				if err != nil {
					t.Errorf("Leave group test failed with response code %d, resp <%s>, err <%s>", status, string(resp), err.Error())
				} else {
					t.Errorf("leave group test failed with response code %d, resp <%s>", status, string(resp))
				}
			}

			//clean group data
			status, resp, err = testnode.RequestAPI(peerapi, "/api/v1/group/clear", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
			if status != 200 {
				if err != nil {
					t.Errorf("clean group test failed with response code %d, resp <%s>, err <%s>", status, string(resp), err.Error())
				} else {
					t.Errorf("clean group test failed with response code %d, resp <%s>", status, string(resp))
				}
			}
		}
	}

}
