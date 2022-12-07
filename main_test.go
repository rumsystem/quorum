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

	groupseed, groupId, err := CreateGroup(ownerNode, groupName)
	if err != nil {
		t.Errorf("create group on owner node failed with error <%s>", err)
		t.Fail()
	}

	logger.Debugf("OK: group {Name <%s>, GroupId<%s>} created on node <%s>", groupName, groupId, ownerNode.NodeName)

	time.Sleep(1 * time.Second)

	logger.Debugf("____________JOIN_GROUP_____________")

	for _, node := range nodes {
		if node.NodeName == OWNER_NODE {
			//skip owner node
			continue
		} else {
			if err = JoinGroup(node, groupseed, groupId); err != nil {
				logger.Warningf("node <%s> join group failed with error <%s>", node.NodeName, groupId, err)
				t.Fail()
			} else {
				logger.Debugf("OK: node <%s> join group <%s> done", node.NodeName, groupId)
			}
			time.Sleep(1 * time.Second)
		}
	}

	//check status of the group on all nodes
	for _, node := range nodes {
		//check group status should be IDLE
		ready := "IDLE"
		groupStatus, err := GetGroupStatus(node, groupId)
		if err != nil {
			t.Fail()
		}

		if groupStatus != ready {
			logger.Errorf("node <%s> group <%s> not idle, status <%s>", node.NodeName, groupId, groupStatus)
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

	ownerEpoch, err := GetEpochOnGroup(ownerNode, groupId)
	if err != nil {
		t.Fail()
	}

	logger.Debugf("Owner epoch <%d>", ownerEpoch)

	for _, node := range nodes {
		if node.NodeName == OWNER_NODE {
			//skip owner node
			continue
		}

		userEpoch, err := GetEpochOnGroup(node, groupId)
		if err != nil {
			t.Fail()
		}

		logger.Debugf("node <%s> epoch <%d>", node.NodeName, userEpoch)
		if ownerEpoch != userEpoch {
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

		if err = VerifyTrxOnGroup(node, groupId, trxs); err != nil {
			t.Fail()
		}
	}

	logger.Debugf("____________LEAVE_GROUP_AND_CLEAR_DATA_____________")
	for _, node := range nodes {

		if err = LeaveGroup(node, groupId); err != nil {
			t.Fail()
		}

		if err = ClearGroup(node, groupId); err != nil {
			t.Fail()
		}
	}
}

/*
How to test

1. create 1 group on owner node
2. owner post 100 trxs
3. user1 join group
4. wait 10s for user1 to finish sync (should be quite enough)
5. check user 1 has same epoch and same trxs
6. user1 send a POST trx
7. verify POST send successful
*/

func TestBasicSync(t *testing.T) {
	logger.Debugf("_____________TestGroupPostContents_RUNNING_____________")

	//owner create a group
	//get owner node
	ownerNode := nodes[OWNER_NODE]
	userNode := nodes[USER_NODE]
	groupName := "testgroup"

	logger.Debugf("_____________CREATE_GROUP_____________")
	groupSeed, groupId, err := CreateGroup(ownerNode, groupName)
	if err != nil {
		t.Fail()
	}

	logger.Debugf("_____________OWNER_POST_TO_GROUP_____________")
	trxs, err := PostToGroup(ownerNode, groupId, 100)
	if err != nil {
		t.Fail()
	}

	logger.Debugf("_____________OWNER_VERIFY_TRXS_____________")
	err = VerifyTrxOnGroup(ownerNode, groupId, trxs)
	if err != nil {
		t.Fail()
	}

	logger.Debugf("_____________USER_JOIN_GROUP_____________")
	err = JoinGroup(userNode, groupSeed, groupId)
	if err != nil {
		t.Fail()
	}

	logger.Debugf("_____________WAIT_USERNODE_SYNC_DONE_____________")
	//wait 10s for user node to finish sync
	time.Sleep(time.Duration(10) * time.Second)

	logger.Debugf("_____________START_VERIFY_____________")
	epochOwner, err := GetEpochOnGroup(ownerNode, groupId)
	if err != nil {
		t.Fail()
	}

	epochUser, err := GetEpochOnGroup(userNode, groupId)
	if err != nil {
		t.Fail()
	}

	//check if user get all epoch
	if epochUser != epochOwner {
		logger.Errorf("User node check epoch failed, highest epoch <%d>, should be <%d>", epochUser, epochOwner)
		t.Fail()
	}

	logger.Debugf("OK: ownernode epoch <%d>, usernode epoch <%d>", epochOwner, epochUser)
	//check user node get all trxs
	err = VerifyTrxOnGroup(userNode, groupId, trxs)
	if err != nil {
		t.Fail()
	}

	logger.Debugf("_____________LEAVE_AND_CLEAR_____________")
	err = LeaveGroup(userNode, groupId)
	if err != nil {
		t.Fail()
	}

	err = ClearGroup(userNode, groupId)
	if err != nil {
		t.Fail()
	}

	err = LeaveGroup(ownerNode, groupId)
	if err != nil {
		t.Fail()
	}

	err = ClearGroup(ownerNode, groupId)
	if err != nil {
		t.Fail()
	}
}

/*
func TestBasicMultiProducerBft(t *testing.T) {
	logger.Debugf("_____________TestBasicMultiProducerBft_RUNNING_____________")
	//owner create a group
	//get owner node
	ownerNode := nodes[OWNER_NODE]
	userNode := nodes[USER_NODE]
	p1node := nodes[PRODUCER_NODE1]
	p2node := nodes[PRODUCER_NODE2]
	groupName := "testgroup"

	groupSeed, groupId, err := CreateGroup(ownerNode, groupName)
	if err != nil {
		t.Fail()
	}

	trxs, err := PostToGroup(ownerNode, groupId, 100)
	if err != nil {
		t.Fail()
	}

	err = VerifyTrxOnGroup(ownerNode, groupId, trxs)
	if err != nil {
		t.Fail()
	}

	err = JoinGroup(userNode, groupSeed, groupId)
	if err != nil {
		t.Fail()
	}

	err = JoinGroup(p1node, groupSeed, groupId)
	if err != nil {
		t.Fail()
	}

	err = JoinGroup(p2node, groupSeed, groupId)
	if err != nil {
		t.Fail()
	}

	//wait 20s for all node to finish sync
	time.Sleep(time.Duration(20) * time.Second)

	epochOwner, err := GetEpochOnGroup(ownerNode, groupId)
	if err != nil {
		t.Fail()
	}

	epochUser, err := GetEpochOnGroup(userNode, groupId)
	if err != nil {
		t.Fail()
	}

	epochp1, err := GetEpochOnGroup(p1node, groupId)
	if err != nil {
		t.Fail()
	}

	epochp2, err := GetEpochOnGroup(p2node, groupId)
	if err != nil {
		t.Fail()
	}

	logger.Debugf("OK: ownernode epoch <%d>, usernode epoch <%d>, p1 epoch <%d>, p2 epoch <%d>",
		epochOwner, epochUser, epochp1, epochp2)

	if epochUser != epochOwner || epochp1 != epochOwner || epochp2 != epochOwner {
		logger.Errorf("User node check epoch failed, highest epoch <%d>, should be <%d>", epochUser, epochOwner)
		t.Fail()
	}

	logger.Debugf("OK: ownernode epoch <%d>, p2 epoch <%d>", epochOwner, epochp2)

	logger.Debugf("_____________LEAVE_AND_CLEAR_____________")

	err = LeaveGroup(userNode, groupId)
	if err != nil {
		t.Fail()
	}

	err = ClearGroup(userNode, groupId)
	if err != nil {
		t.Fail()
	}

	err = LeaveGroup(ownerNode, groupId)
	if err != nil {
		t.Fail()
	}

	err = ClearGroup(ownerNode, groupId)
	if err != nil {
		t.Fail()
	}

	err = LeaveGroup(p1node, groupId)
	if err != nil {
		t.Fail()
	}

	err = ClearGroup(p1node, groupId)
	if err != nil {
		t.Fail()
	}

	err = LeaveGroup(p2node, groupId)
	if err != nil {
		t.Fail()
	}

	err = ClearGroup(p2node, groupId)
	if err != nil {
		t.Fail()
	}
}

*/

func CreateGroup(node *testnode.NodeInfo, groupName string) (groupseed, groupId string, err error) {
	logger.Debugf("node <%s> try create group with name <%s>", node.NodeName, groupName)
	status, resp, err := testnode.RequestAPI(node.APIBaseUrl, "/api/v1/group", "POST", fmt.Sprintf(`{"group_name":"%s","app_key":"default", "consensus_type":"poa","encryption_type":"public"}`, groupName))
	if err == nil || status != 200 {
		var objmap map[string]interface{}
		if err := json.Unmarshal(resp, &objmap); err != nil {
			logger.Errorf("Data Unmarshal error %s", err)
			return "", "", err
		} else {
			groupseed = string(resp)
			seedurl := objmap["seed"]
			groupId = testnode.SeedUrlToGroupId(seedurl.(string))
			logger.Debugf("OK: group {Name <%s>, GroupId<%s>} created on node <%s>", groupName, groupId, node.NodeName)
		}
	} else {
		logger.Errorf("create group on owner node failed with error <%s>", err)
		return "", "", err
	}
	return groupseed, groupId, nil
}

func JoinGroup(node *testnode.NodeInfo, seed string, groupId string) error {
	logger.Debugf("node <%s> try join group with groupId <%s>", node.NodeName, groupId)
	_, _, err := testnode.RequestAPI(node.APIBaseUrl, "/api/v2/group/join", "POST", seed)
	if err != nil {
		logger.Warningf("node <%s> join group failed with error <%s>", node.NodeName, groupId, err)
		return err
	} else {
		logger.Debugf("OK: node <%s> join group <%s> done", node.NodeName, groupId)
		return nil
	}
}

func PostToGroup(node *testnode.NodeInfo, groupId string, trxNum int) (map[string]string, error) {
	logger.Debugf("node <%s> try send <%d> trxs to  group with groupId <%s>", node.NodeName, trxNum, groupId)
	trxs := make(map[string]string) //trx_id: trx_content

	i := 0
	for i < trxNum {
		var postContent string
		var resp []byte
		var err error

		r := GetGussRandNum(int64(randRangeMin), int64(randRangeMax)) // from 10ms (0.01s) to 500ms (1s)
		logger.Debugf("node <%s> try post trx", node.NodeName)
		postContent = fmt.Sprintf(`{"type":"Add","object":{"type":"Note","content":"post_content_from_%s_%d","name":"%s_%d"},"target":{"id":"%s","type":"Group"}}`,
			node.NodeName, i,
			node.NodeName, i,
			groupId)
		_, resp, err = testnode.RequestAPI(node.APIBaseUrl, "/api/v1/group/content/false", "POST", postContent)

		if err != nil {
			logger.Errorf("post content to api error %s", err)
			return nil, err
		}

		var objmap map[string]interface{}
		if err = json.Unmarshal(resp, &objmap); err != nil {
			// store trx id, verify it later on each group
			logger.Errorf("Data Unmarshal error %s", err)
			return nil, err
		}

		if objmap["trx_id"] != nil {
			logger.Debugf("OK: post with trxid: <%s>", objmap["trx_id"].(string))
			trxs[objmap["trx_id"].(string)] = "posted"
		} else {
			logger.Errorf("resp body was not included trx_id %s", string(resp))
			err = fmt.Errorf("resp error")
			return nil, err
		}

		time.Sleep(time.Duration(r) * time.Millisecond)
		i++
	}
	return trxs, nil
}

func VerifyTrxOnGroup(node *testnode.NodeInfo, groupId string, trxs map[string]string) error {
	for trxId, _ := range trxs {
		logger.Debugf("check trx <%s>", trxId)
		_, resp, err := testnode.RequestAPI(node.APIBaseUrl, fmt.Sprintf("/api/v1/trx/%s/%s", groupId, trxId), "GET", "")
		if err == nil {
			var data map[string]interface{}
			if err := json.Unmarshal(resp, &data); err != nil {
				logger.Errorf("Data Unmarshal error %s", err)
				return err
			}
			//TBD, run more check on trx
			if data["TrxId"] == trxId {
				logger.Debugf("Ok: ---- node <%s> trx <%s> verified", node.NodeName, trxId)
			} else {
				logger.Errorf("trx <%s> verify failed on node <%s>", trxId, node.NodeName)
				err = fmt.Errorf("node <%s> verify trx <%s> failed", node.NodeName, trxId)
				return err
			}
		} else {
			err = fmt.Errorf("get /api/v1/trx/%s err: %s", trxId, err)
			logger.Error(err.Error())
			return err
		}
	}

	return nil
}

func GetEpochOnGroup(node *testnode.NodeInfo, groupId string) (epoch int, err error) {
	logger.Debugf("node <%s> get epoch on group with groupId <%s>", node.NodeName, groupId)

	_, resp, err := testnode.RequestAPI(node.APIBaseUrl, "/api/v1/groups", "GET", "")
	if err != nil {
		logger.Errorf("node <%s> get group info failed with error <%s>", node.NodeName, err.Error())
		return -1, err
	}

	groupslist := &api.GroupInfoList{}
	if err := json.Unmarshal(resp, &groupslist); err != nil {
		logger.Errorf("parse peer group error %s", err)
		return -1, err
	}

	groupInfo := groupslist.GroupInfos[0]
	return int(groupInfo.Epoch), nil
}

func GetGroupStatus(node *testnode.NodeInfo, groupId string) (string, error) {
	logger.Debugf("node <%s> get groupstatus on group with groupId <%s>", node.NodeName, groupId)

	_, resp, err := testnode.RequestAPI(node.APIBaseUrl, "/api/v1/groups", "GET", "")
	if err != nil {
		logger.Errorf("node <%s> get group info failed with error <%s>", node.NodeName, err.Error())
		return "", err
	}

	groupslist := &api.GroupInfoList{}
	if err := json.Unmarshal(resp, &groupslist); err != nil {
		logger.Errorf("parse peer group error %s", err)
		return "", err
	}

	groupInfo := groupslist.GroupInfos[0]
	return groupInfo.GroupStatus, nil
}

func LeaveGroup(node *testnode.NodeInfo, groupId string) error {
	//leave group
	status, resp, err := testnode.RequestAPI(node.APIBaseUrl, "/api/v1/group/leave", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
	if err != nil {
		logger.Errorf("Leave group test failed with response code %d, resp <%s>, err <%s>", status, string(resp), err.Error())
		return err
	} else if status != 200 {
		err = fmt.Errorf("leave group test failed with response code %d, resp <%s>", status, string(resp))
		logger.Errorf(err.Error())
		return err
	} else {
		logger.Debugf("OK: node <%s> leave group <%s>", node.NodeName, groupId)
	}

	return nil
}

func ClearGroup(node *testnode.NodeInfo, groupId string) error {
	//clean group data
	status, resp, err := testnode.RequestAPI(node.APIBaseUrl, "/api/v1/group/clear", "POST", fmt.Sprintf(`{"group_id":"%s"}`, groupId))
	if err != nil {
		logger.Errorf("clean group test failed with response code %d, resp <%s>, err <%s>", status, string(resp), err.Error())
		return err
	} else if status != 200 {
		err = fmt.Errorf("clean group test failed with response code %d, resp <%s>", status, string(resp))
		logger.Errorf(err.Error())
		return err
	} else {
		logger.Debugf("OK : node <%s> clear group date <%s>", node.NodeName, groupId)
	}

	return nil
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
