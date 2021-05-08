package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/huo-ju/quorum/internal/pkg/api"
	"github.com/huo-ju/quorum/testnode"
	"log"
	"os"
	"testing"
	"time"
)

var pidlist []int
var bootstrapapi, peer1api, peer2api string

func TestMain(m *testing.M) {

	log.Println("Setup testing nodes")
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

	var tempdatadir string
	bootstrapapi, peer1api, peer2api, tempdatadir, _ = testnode.Run2NodeProcessWith1Bootstrap(context.Background(), pidch)
	exitVal := m.Run()
	log.Println("after tests clean:", tempdatadir)
	testnode.Cleanup(tempdatadir, pidlist)
	os.Exit(exitVal)
}

func TestNodeStatus(t *testing.T) {
	expected := "NODE_ONLINE"
	for _, peerapi := range []string{peer1api, peer2api} {
		resp, err := testnode.RequestAPI(peerapi, "/api/v1/node", "GET", "")
		if err == nil {
			var objmap map[string]interface{}
			if err := json.Unmarshal(resp, &objmap); err != nil {
				t.Errorf("Data Unmarshal error %s", err)
			} else {
				if objmap["node_status"] != expected {
					t.Fail()
					t.Logf("Expected %s, got %s", expected, objmap["node_status"])
				} else {
					t.Logf("api %s status: %s", peerapi, objmap["node_status"])
				}
			}
		} else {
			t.Errorf("api request error %s", err)
		}
	}
}

func TestGroups(t *testing.T) {

	//create 5 groups on peer1, and another 5 groups on peer2, and join all groups, then verify peer1 groups == peer2 groups

	var genesisblockpeer1 []string
	var genesisblockpeer2 []string

	groupspeernum := 5

	for i := 0; i < groupspeernum; i++ {
		resp, err := testnode.RequestAPI(peer1api, "/api/v1/group", "POST", fmt.Sprintf(`{"group_name":"testgroup_peer_%d_%d"}`, 1, i))
		if err == nil {
			var objmap map[string]interface{}
			if err := json.Unmarshal(resp, &objmap); err != nil {
				t.Errorf("Data Unmarshal error %s", err)
			} else {
				genesisblockpeer1 = append(genesisblockpeer1, string(resp))
				group_name := objmap["group_name"]
				log.Printf("group %s created on peer%d", group_name, 1)
			}
		} else {
			t.Errorf("create group on peer%d error %s", 1, err)
		}
		resp, err = testnode.RequestAPI(peer2api, "/api/v1/group", "POST", fmt.Sprintf(`{"group_name":"testgroup_peer_%d_%d"}`, 2, i))
		if err == nil {
			var objmap map[string]interface{}
			if err := json.Unmarshal(resp, &objmap); err != nil {
				t.Errorf("Data Unmarshal error %s", err)
			} else {
				genesisblockpeer2 = append(genesisblockpeer2, string(resp))
				group_name := objmap["group_name"]
				log.Printf("group %s created on peer%d", group_name, 2)
			}
		} else {
			t.Errorf("create group on peer%d error %s", 2, err)
		}
	}

	if len(genesisblockpeer1) == groupspeernum && len(genesisblockpeer2) == groupspeernum {
		for i := 0; i < groupspeernum; i++ {
			g1 := genesisblockpeer1[i]
			g2 := genesisblockpeer2[i]
			_, err := testnode.RequestAPI(peer1api, "/api/v1/group/join", "POST", g2)
			if err != nil {
				t.Errorf("peer1 join group %d error %s", i, err)
			}
			_, err = testnode.RequestAPI(peer2api, "/api/v1/group/join", "POST", g1)
			if err != nil {
				t.Errorf("peer2 join group %d error %s", i, err)
			}
		}

		ready := "GROUP_READY"
		syncfinish := false
		waitingcounter := 0
		for {
			if waitingcounter >= 5 {
				break
			}
			groupslist1 := &api.GroupInfoList{}
			groupslist2 := &api.GroupInfoList{}
			resp, err := testnode.RequestAPI(peer1api, "/api/v1/group", "GET", "")
			if err == nil {
				if err := json.Unmarshal(resp, &groupslist1); err != nil {
					t.Errorf("get peer1 group  error %s", err)
				}
			}

			resp, err = testnode.RequestAPI(peer2api, "/api/v1/group", "GET", "")
			if err == nil {
				if err := json.Unmarshal(resp, &groupslist2); err != nil {
					t.Errorf("get peer2 group  error %s", err)
				}
			}
			syncfinish = true
			if len(groupslist1.GroupInfos) == 2*groupspeernum && len(groupslist2.GroupInfos) == 2*groupspeernum {
				log.Printf("%d/%d groups on peer1/peer2", len(groupslist1.GroupInfos), len(groupslist2.GroupInfos))
				for _, groupinfo := range groupslist1.GroupInfos {
					if groupinfo.GroupStatus != ready {
						syncfinish = false
						t.Logf("peer1 %s status: %s ", groupinfo.GroupName, groupinfo.GroupStatus)
					}
				}
				for _, groupinfo := range groupslist2.GroupInfos {
					if groupinfo.GroupStatus != ready {
						syncfinish = false
						t.Logf("peer2 %s status: %s ", groupinfo.GroupName, groupinfo.GroupStatus)
					}
				}
			} else {
				t.Errorf("Expected %d/%d groups on peer1/peer2, got %d/%d", 2*groupspeernum, 2*groupspeernum, len(groupslist1.GroupInfos), len(groupslist2.GroupInfos))
			}
			if syncfinish == true {
				break
			} else {
				log.Printf("waiting 10 seconds for peers data sync")
				time.Sleep(10 * time.Second)
			}
			waitingcounter += 1
		}

		if syncfinish == false {
			t.Errorf("error: peer data sync not finish. ")
		}
	} else {
		t.Fail()
		t.Logf("Expected %d groups on peer1/peer2 got %d/%d", groupspeernum, len(genesisblockpeer1), len(genesisblockpeer2))
	}

	//resp, err := testnode.RequestAPI(apiurl, "/group", "POST", )
	//time.Sleep(10000 * time.Minute)
}
