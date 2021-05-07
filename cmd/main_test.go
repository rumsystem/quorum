package main

import (
	"context"
	"encoding/json"
	"github.com/huo-ju/quorum/testnode"
	"log"
	"os"
	"testing"
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
