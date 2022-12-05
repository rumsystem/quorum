package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/pkg/chainapi/api"
	"github.com/rumsystem/quorum/testnode"

	"time"
)

/*
	To Verify :
		Node create a group and leave it, should be able to rejoin and group
*/

type RespError struct {
	Error string `json:"error"`
}

// temp solution
const (
	BOOTSTRAP      = "bootstrap"
	OWNER_NODE     = "fullnode_0"
	USER_NODE      = "fullnode_1"
	PRODUCER_NODE1 = "producernode_0"
	PRODUCER_NODE2 = "producernode_1"
)

var (
	pidlist                                                                       []int
	bootstrapNode                                                                 *testnode.NodeInfo
	nodes                                                                         map[string]*testnode.NodeInfo
	fullnodes, producernodes, groups, posts, synctime, randRangeMin, randRangeMax int
	logger                                                                        = logging.Logger("main_test")
)

func TestMain(m *testing.M) {

	logging.SetLogLevel("main_test", "debug")

	producerNodesPtr := flag.Int("bpnodes", 2, "mock producernodes")
	fullnodesPtr := flag.Int("fullnodes", 2, "mock fullnodes")
	groupsPtr := flag.Int("groups", 1, "groups on owner node")
	postsPtr := flag.Int("posts", 100, "posts to group")
	synctimePtr := flag.Int("synctime", 3, "time to wait before verify")
	rextestmode := flag.Bool("rextest", false, "RumExchange Test Mode")
	randMin := flag.Int("rmin", 10, "post rand min value")
	randMax := flag.Int("rmax", 200, "post rand max value")

	flag.Parse()

	fullnodes = *fullnodesPtr
	producernodes = *producerNodesPtr
	groups = *groupsPtr
	posts = *postsPtr
	synctime = *synctimePtr
	randRangeMin = *randMin
	randRangeMax = *randMax

	if randRangeMax-randRangeMin < 100 {
		logger.Error("different between randMin / randMax should larger than 100(ms)")
		return
	}

	logger.Debugf("Setup test env")
	logger.Debugf(">>> full nodes: <%d>", fullnodes)
	logger.Debugf(">>> producer nodes: <%d>", producernodes)
	logger.Debugf(">>> group <%d>", groups)
	pidch := make(chan int)

	go func() {
		for {
			select {
			case pid := <-pidch:
				logger.Debug("receive pid", pid)
				pidlist = append(pidlist, pid)
				if len(pidlist) == len(nodes) {
					logger.Debugf("All done...")
					return
				}
			}
		}
	}()

	cliargs := testnode.Nodecliargs{Rextest: *rextestmode}
	var tempdatadir string

	//bootstrapapi, peerapilist, tempdatadir, _ = testnode.RunNodesWithBootstrap(context.Background(), cliargs, pidch, fullnodes, bpnodes)

	nodelist, tempdatadir, _ := testnode.RunNodesWithBootstrap(context.Background(), cliargs, pidch, fullnodes, producernodes)

	nodes = make(map[string]*testnode.NodeInfo)
	//transfer list to map
	for _, node := range nodelist {
		if node.NodeName == BOOTSTRAP {
			bootstrapNode = node
		} else {
			nodes[node.NodeName] = node
		}
	}

	exitVal := m.Run()
	logger.Debug("all tests finished, clean up:", tempdatadir)
	testnode.Cleanup(tempdatadir, nodelist)
	os.Exit(exitVal)
}

func TestJoinGroup(t *testing.T) {
	logger.Debugf("_____________TestJoinGroup_RUNNING_____________")

	//get owner node
	ownerNode := nodes[OWNER_NODE]

	//create 3 group on fullnode, join the group then leave, repeat 3 times and verify the group exist and in "IDLE" status
	groupToCreate := 3
	count := 0
	for count < 3 {
		for i := 0; i < groupToCreate; i++ {
			logger.Debugf("_____________CREATE_GROUP_____________")
			var groupseed string
			var groupId string
			groupName := fmt.Sprintf("testgroup_%d", i+1)
			status, resp, err := testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v1/group", "POST", fmt.Sprintf(`{"group_name":"%s","app_key":"default", "consensus_type":"poa","encryption_type":"public"}`, groupName))
			if err == nil || status != 200 {
				var objmap map[string]interface{}
				if err := json.Unmarshal(resp, &objmap); err != nil {
					t.Errorf("Data Unmarshal error %s", err)
				} else {
					groupseed = string(resp)
					seedurl := objmap["seed"]
					groupId = testnode.SeedUrlToGroupId(seedurl.(string))
					logger.Debugf("group {Name <%s>, GroupId<%s>} created on node <%s>", groupName, groupId, ownerNode.NodeName)
				}
			} else {
				t.Errorf("create group on peer%d error %s", 1, err)
			}
			time.Sleep(1 * time.Second)
			// try join the same group just created
			logger.Debugf("_____________TEST_JOIN_EXIST_GROUP_____________")
			status, _, err = testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v2/group/join", "POST", groupseed)

			//check if failed
			if status != 400 {
				t.Errorf("Join existed group test failed with err %s", err.Error())
			}

			time.Sleep(1 * time.Second)

			logger.Debugf("_____________TEST_LEAVE_GROUP_____________")
			status, _, _ = testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v1/group/leave", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
			if status != 200 {
				t.Errorf("Leave group test failed with response code %d", status)
			}
			time.Sleep(1 * time.Second)

			logger.Debugf("_____________TEST_JOIN_LEAVED_GROUP_____________")
			status, _, _ = testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v2/group/join", "POST", groupseed)
			if status != 200 {
				t.Errorf("join leaved group test failed with response code %d", status)
			}
			// verify group status
			_, resp, err = testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v1/groups", "GET", "")

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

			//ready := "IDLE"
			for _, groupinfo := range groupslist.GroupInfos {
				logger.Debugf("Group %s status %s", groupinfo.GroupId, groupinfo.GroupStatus)
				if groupinfo.GroupId != groupId {
					t.Errorf("Check group status failed %s, groupId mismatch", err)
				}
				//No need to check IDLE status in this test case
				//if groupinfo.GroupStatus != ready {
				//	t.Errorf("Check group status failed %s, group not IDLE", err)
				//}
			}

			logger.Debugf("_____________TEST_LEAVE_GROUP_____________")
			//leave group
			status, resp, err = testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v1/group/leave", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
			if status != 200 {
				if err != nil {
					t.Errorf("Leave group test failed with response code %d, resp <%s>, err <%s>", status, string(resp), err.Error())
				} else {
					t.Errorf("leave group test failed with response code %d, resp <%s>", status, string(resp))
				}
			}

			logger.Debugf("_____________TEST_CLEAR_GROUP_____________")
			//clear group data
			status, resp, err = testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v1/group/clear", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
			if status != 200 {
				if err != nil {
					t.Errorf("clean group test failed with response code %d, resp <%s>, err <%s>", status, string(resp), err.Error())
				} else {
					t.Errorf("clean group test failed with response code %d, resp <%s>", status, string(resp))
				}
			}
		}

		time.Sleep(1 * time.Second)
		count++
	}
}

/*
How to test:

1. create 1 owner node
2. create 1 user node
3. create 2 producer nodes
4. create 1 group on owner node
5. all other nodes join the group
6. owner node and user node send POST TRX in time gap following normal distribution(nora)
7. check all nodes has the same block epoch at the end of test
8. verify owner node and user node has the same trx_id list at the end of test (since producer node DOES NOT apply POST trx)
9. clean up
*/
func TestGroupPostContents(t *testing.T) {

	logger.Debugf("_____________TestGroupPostContents_RUNNING_____________")
	logger.Debugf("_____________CREATE_GROUP_____________")

	var groupseed string
	var groupId string

	//owner create a group
	//get owner node
	ownerNode := nodes[OWNER_NODE]
	userNode := nodes[USER_NODE]
	groupName := "testgroup"

	status, resp, err := testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v1/group", "POST", fmt.Sprintf(`{"group_name":"%s","app_key":"default", "consensus_type":"poa","encryption_type":"public"}`, groupName))
	if err == nil || status != 200 {
		var objmap map[string]interface{}
		if err := json.Unmarshal(resp, &objmap); err != nil {
			t.Errorf("Data Unmarshal error %s", err)
		} else {
			groupseed = string(resp)
			seedurl := objmap["seed"]
			groupId = testnode.SeedUrlToGroupId(seedurl.(string))
			logger.Debugf("OK: group {Name <%s>, GroupId<%s>} created on node <%s>", groupName, groupId, ownerNode.NodeName)
		}
	} else {
		t.Errorf("create group on owner node failed with error <%s>", err)
		t.Fail()
	}

	time.Sleep(1 * time.Second)

	logger.Debugf("____________JOIN_GROUP_____________")

	for _, node := range nodes {
		if node.NodeName == OWNER_NODE {
			//skip owner node
			continue
		} else {
			logger.Debugf("node <%s> try join group <%s>", node.NodeName, groupId)
			_, _, err := testnode.RequestAPI(node.APIBaseUrl, "/api/v2/group/join", "POST", groupseed)
			if err != nil {
				logger.Warningf("node <%s> join group failed with error <%s>", node.NodeName, groupId, err)
				t.Fail()
			} else {
				logger.Debugf("OK: node <%s> join group <%s> done", node.NodeName, groupId)
			}
		}
		time.Sleep(1 * time.Second)
	}

	//check status of the group on all nodes
	for _, node := range nodes {
		_, resp, err := testnode.RequestAPI(node.APIBaseUrl, "/api/v1/groups", "GET", "")
		if err != nil {
			logger.Errorf("node <%s> get group info failed with error <%s>", node.NodeName, err.Error())
			t.Fail()
		}

		groupslist := &api.GroupInfoList{}
		if err := json.Unmarshal(resp, &groupslist); err != nil {
			logger.Errorf("parse peer group error %s", err)
			t.Fail()
		}

		//check there should be only 1 group on each node and the groupid should be the same
		if len(groupslist.GroupInfos) != 1 || groupslist.GroupInfos[0].GroupId != groupId {
			logger.Errorf("node <%s> group number/groupid mistch match, group count <%d>", node.NodeName, len(groupslist.GroupInfos))
			t.Fail()
		}

		//check group status should be IDLE
		ready := "IDLE"
		groupInfo := groupslist.GroupInfos[0]
		if groupInfo.GroupStatus != ready {
			logger.Errorf("node <%s> group <%s> not idle, status <%s>", node.NodeName, groupId, groupInfo.GroupStatus)
			t.Fail()
		} else {
			logger.Debugf("OK: node <%s> group <%s> ready", node.NodeName, groupId)
		}
	}

	logger.Debugf("____________START_POST_____________")
	//send trxs randomly

	logger.Debugf("Try post <%s> trxs", posts)
	trxs := make(map[string]string) //trx_id: trx_content
	i := 0
	for i < posts {
		var postContent string
		var resp []byte
		var err error
		r := GetGussRandNum(int64(randRangeMin), int64(randRangeMax)) // from 10ms (0.01s) to 500ms (1s)
		if r%2 == 0 {
			logger.Debugf("owner node try post trx")
			postContent = fmt.Sprintf(`{"type":"Add","object":{"type":"Note","content":"post_content_from_%s_%d","name":"%s_%d"},"target":{"id":"%s","type":"Group"}}`, OWNER_NODE, i, OWNER_NODE, i, groupId)
			_, resp, err = testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v1/group/content/false", "POST", postContent)

		} else {
			logger.Debugf("user node try post trx")
			postContent = fmt.Sprintf(`{"type":"Add","object":{"type":"Note","content":"post_content_from_%s_%d","name":"%s_%d"},"target":{"id":"%s","type":"Group"}}`, USER_NODE, i, USER_NODE, i, groupId)
			_, resp, err = testnode.RequestAPI(userNode.APIBaseUrl, "/api/v1/group/content/false", "POST", postContent)
		}

		if err != nil {
			logger.Errorf("post content to api error %s", err)
			t.Fail()
		}

		var objmap map[string]interface{}
		if err = json.Unmarshal(resp, &objmap); err != nil {
			// store trx id, verify it later on each group
			logger.Errorf("Data Unmarshal error %s", err)
			t.Fail()
		}

		if objmap["trx_id"] != nil {
			logger.Debugf("OK: post with trxid: <%s>", objmap["trx_id"].(string))
			trxs[objmap["trx_id"].(string)] = "posted"
		} else {
			logger.Errorf("resp body was not included trx_id %s", string(resp))
			t.Fail()
		}

		logger.Debugf("sleep %d ms ", r)
		time.Sleep(time.Duration(r) * time.Millisecond)
		i++
	}

	//sleep 5 seconds to make sure all node received latest block
	time.Sleep(2 * time.Second)

	logger.Debugf("____________VERIFY_EPOCH_____________")

	//verify all nodes has same block epoch
	//owenr node get group epoch
	_, resp, err = testnode.RequestAPI(ownerNode.APIBaseUrl, "/api/v1/groups", "GET", "")
	if err != nil {
		logger.Errorf("node <%s> get group info failed with error <%s>", ownerNode.NodeName, err.Error())
		t.Fail()
	}

	groupslist := &api.GroupInfoList{}
	if err := json.Unmarshal(resp, &groupslist); err != nil {
		logger.Errorf("parse peer group error %s", err)
		t.Fail()
	}

	owenrGroupInfo := groupslist.GroupInfos[0]
	logger.Debugf("Owner epoch <%d>", owenrGroupInfo.Epoch)

	for _, node := range nodes {
		if node.NodeName == OWNER_NODE {
			//skip owner node
			continue
		}
		_, resp, err := testnode.RequestAPI(node.APIBaseUrl, "/api/v1/groups", "GET", "")
		if err != nil {
			logger.Errorf("node <%s> get group info failed with error <%s>", node.NodeName, err.Error())
			t.Fail()
		}

		groupslist := &api.GroupInfoList{}
		if err := json.Unmarshal(resp, &groupslist); err != nil {
			logger.Errorf("parse peer group error %s", err)
			t.Fail()
		}

		groupInfo := groupslist.GroupInfos[0]
		logger.Debugf("node <%s> epoch <%d>", node.NodeName, groupInfo.Epoch)
		if groupInfo.Epoch != owenrGroupInfo.Epoch {
			logger.Errorf("node <%s> epoch mismatch", node.NodeName)
			t.Fail()
		} else {
			logger.Debugf("OK: node <%s> get latest block", node.NodeName)
		}
	}

	logger.Debugf("____________VERIFY_TRX_____________")

	//verify all nodes (except producer1 and producer2) have all trxs sent by owner and user
	for _, node := range nodes {
		if node.NodeName == PRODUCER_NODE1 || node.NodeName == PRODUCER_NODE2 {
			continue
		}
		logger.Debugf(">>>>>>> node <%s>", node.NodeName)
		for trxId, _ := range trxs {
			logger.Debugf("check trx <%s>", trxId)
			_, resp, err := testnode.RequestAPI(node.APIBaseUrl, fmt.Sprintf("/api/v1/trx/%s/%s", groupId, trxId), "GET", "")
			if err == nil {
				var data map[string]interface{}
				if err := json.Unmarshal(resp, &data); err != nil {
					logger.Errorf("Data Unmarshal error %s", err)
					t.Fail()
				}
				//TBD, run more check on trx
				if data["TrxId"] == trxId {
					logger.Debugf("Ok: ---- trx <%s> verified,", trxId)
				} else {
					logger.Debugf("????")
				}
			} else {
				logger.Errorf("get /api/v1/trx/%s err: %s", trxId, err)
				t.Fail()
			}
		}
	}

	/*



		ready := "IDLE"

		for i := 0; i < fullnodes+bpnodes; i++ {
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
				_, resp, err := testnode.RequestAPI(peer1api, "/api/v1/group/content/false", "POST", content)
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
				logger.Debugf("sleep: %.2f s before next post\n", sleepTime)
				time.Sleep(time.Duration(sleepTime*1000) * time.Millisecond)
				//time.Sleep(time.Duration(5*1000) * time.Millisecond)
			}
		}
		t.Logf("waiting %d seconds for peers data sync", synctime)
		time.Sleep(time.Duration(synctime) * time.Second)
		logger.Debug("start verify groups content")

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

				t.Logf("start verify node%d, group id: %s", nodeIdx+1, groupId)
				_, resp, err := testnode.RequestAPI(peerapi, fmt.Sprintf("/app/api/v1/group/%s/content?num=100", groupId), "POST", "{\"senders\":[]}")
				groupcontentlist := []appapi.ContentStruct{}

				if err == nil {
					if err := json.Unmarshal(resp, &groupcontentlist); err != nil {
						print(string(resp))
						t.Errorf("Data Unmarshal error %s", err)
					}
				} else {
					t.Errorf("get /api/v1/group/content err: %s", err)
				}
				for _, contentitem := range groupcontentlist {
					if contentitem.Content != nil {
						if _, found := trxStatus[contentitem.TrxId]; found {
							trxStatus[contentitem.TrxId] = true
							t.Logf("trx %s ok", contentitem.TrxId)
						} else {
							t.Errorf("trx %s not exists in this groups", contentitem.TrxId)
						}
					}
				}

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

	*/

}

// Box muller
func GetGussRandNum(min, max int64) int64 {
	sigma := (float64(min) + float64(max)) / 2
	miu := (float64(max) - sigma) / 3
	rand.Seed(time.Now().UnixNano())
	x := rand.Float64()
	x1 := rand.Float64()
	a := math.Cos(2*math.Pi*x) * math.Sqrt((-2)*math.Log(x1))
	result := a*miu + sigma
	return int64(result)
}
