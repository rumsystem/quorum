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
	for _, peerapi := range []string{peer1api, peer2api} {
		if peerapi == "" {
			t.Fail()
			t.Logf("peerapi should not be nil.")
		}
		t.Logf("request API at: %s", peerapi)
		resp, err := testnode.RequestAPI(peerapi, "/api/v1/node", "GET", "")
		if err == nil {
			var objmap map[string]interface{}
			if err := json.Unmarshal(resp, &objmap); err != nil {
				t.Errorf("Data Unmarshal error %s", err)
			} else {
				if objmap["node_publickey"] == "" {
					t.Fail()
					t.Logf("Expected node publickey not nil")
				} else {
					t.Logf("api %s status ok", peerapi)
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
		time.Sleep(5 * time.Second)
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
		waitingcounter := 0
		groupStatus := make(map[string]bool) // add ready groups
		for {
			if waitingcounter >= 10 {
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
			if len(groupslist1.GroupInfos) == 2*groupspeernum && len(groupslist2.GroupInfos) == 2*groupspeernum {
				log.Printf("%d/%d groups on peer1/peer2", len(groupslist1.GroupInfos), len(groupslist2.GroupInfos))
				for _, groupinfo := range groupslist1.GroupInfos {
					if groupinfo.GroupStatus != ready {
						t.Logf("peer1 %s status: %s ", groupinfo.GroupName, groupinfo.GroupStatus)
					} else {
						t.Logf("peer1 %s status: %s ", groupinfo.GroupName, groupinfo.GroupStatus)
						groupStatus[groupinfo.GroupName] = true
					}
				}
				for _, groupinfo := range groupslist2.GroupInfos {
					if groupinfo.GroupStatus != ready {
						t.Logf("peer2 %s status: %s ", groupinfo.GroupName, groupinfo.GroupStatus)
					} else {
						t.Logf("peer2 %s status: %s ", groupinfo.GroupName, groupinfo.GroupStatus)
						groupStatus[groupinfo.GroupName] = true
					}
				}
			} else {
				t.Fail()
				t.Errorf("Expected %d/%d groups on peer1/peer2, got %d/%d", 2*groupspeernum, 2*groupspeernum, len(groupslist1.GroupInfos), len(groupslist2.GroupInfos))
			}
			if len(groupStatus) == 10 {
				break
			} else {
				log.Printf("waiting 30 seconds for peers data sync")
				time.Sleep(30 * time.Second)
			}
			waitingcounter += 1
		}

		if len(groupStatus) != 10 {
			t.Errorf("error: peer data sync not finish. ")
		}
	} else {
		t.Fail()
		t.Logf("Expected %d groups on peer1/peer2 got %d/%d", groupspeernum, len(genesisblockpeer1), len(genesisblockpeer2))
	}
}

func TestGroupsContent(t *testing.T) {

	//create 5 posts on each groups, then verify peer1 groups have the same posts with peer2 groups

	groupslist1 := &api.GroupInfoList{}
	groupslist2 := &api.GroupInfoList{}
	resp, err := testnode.RequestAPI(peer1api, "/api/v1/group", "GET", "")
	if err == nil {
		if err := json.Unmarshal(resp, &groupslist1); err != nil {
			t.Errorf("Data Unmarshal error %s", err)
		}
	} else {
		t.Errorf("request api /api/v1/group err: %s", err)
	}
	resp, err = testnode.RequestAPI(peer2api, "/api/v1/group", "GET", "")
	if err == nil {
		if err := json.Unmarshal(resp, &groupslist2); err != nil {
			t.Errorf("Data Unmarshal error %s", err)
		}
	} else {
		t.Errorf("request api /api/v1/group err: %s", err)
	}

	for _, groupinfo := range groupslist1.GroupInfos {
		log.Println("post content to each groups")
		i := 1
		for ; i <= 5; i++ {
			content := fmt.Sprintf(`{"type":"Add","object":{"type":"Note","content":"peer1_content_%s_%d","name":"peer1_name_%s_%d"},"target":{"id":"%s","type":"Group"}}`, groupinfo.GroupId, i, groupinfo.GroupId, i, groupinfo.GroupId)
			log.Println(content)
			resp, err := testnode.RequestAPI(peer1api, "/api/v1/group/content", "POST", content)
			if err != nil {
				t.Errorf("post content to api error %s", err)
			}
			var objmap map[string]interface{}
			if err = json.Unmarshal(resp, &objmap); err != nil {
				t.Errorf("Data Unmarshal error %s", err)
			}
			time.Sleep(5 * time.Second)
		}
	}

	log.Println("waiting 60 seconds for peers data sync")
	time.Sleep(60 * time.Second)
	log.Println("start verify groups content")
	for _, groupinfo := range groupslist1.GroupInfos {
		contentlist := make(map[string]string)
		groupcontentlist1 := []api.GroupContentObjectItem{}
		log.Printf("get peer1 group %s  content", groupinfo.GroupId)
		reqdata := fmt.Sprintf(`{"group_id":"%s"}`, groupinfo.GroupId)
		resp, err := testnode.RequestAPI(peer1api, "/api/v1/group/content", "GET", reqdata)

		if err == nil {
			if err := json.Unmarshal(resp, &groupcontentlist1); err != nil {
				t.Errorf("Data Unmarshal error %s", err)
			}
		} else {
			t.Errorf("get /api/v1/group/content err: %s", err)
		}

		for _, contentitem := range groupcontentlist1 {
			if contentitem.Content != nil {
				contentlist[contentitem.TrxId] = contentitem.Content.Content
			}
		}

		log.Printf("peer1 group %s content number: %d", groupinfo.GroupId, len(contentlist))
		//verify with peer2

		groupcontentlist2 := []api.GroupContentObjectItem{}
		resp, err = testnode.RequestAPI(peer2api, "/api/v1/group/content", "GET", reqdata)

		if err == nil {
			if err := json.Unmarshal(resp, &groupcontentlist2); err != nil {
				t.Errorf("Data Unmarshal error %s", err)
			}
		} else {
			t.Errorf("get /api/v1/group/content err: %s", err)
		}

		contentcount := 0
		for _, contentitem := range groupcontentlist2 {
			if contentitem.Content != nil {
				if contentlist[contentitem.TrxId] == contentitem.Content.Content {
					log.Printf("trxid: %s find in peer2's group.\n", contentitem.TrxId)
					contentcount++
				}
			}
		}
		if contentcount == len(contentlist) {
			log.Printf("group %s content check ok.", groupinfo.GroupId)
		} else {
			t.Fail()
			t.Logf("Expected groups %s content number %d, got %d", groupinfo.GroupId, len(contentlist), contentcount)
		}
	}
}
