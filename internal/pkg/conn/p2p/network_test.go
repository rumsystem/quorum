//go:build !js
// +build !js

package p2p

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/rumsystem/quorum/testnode"
)

func TestNodeConnecting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockRendezvousString := "testrendezvousuniqstring"
	bootstrap, peer1, peer2, _, _, _, err := testnode.Run2nodes(ctx, mockRendezvousString)
	if err != nil {
		t.Errorf("create test peers err:%s", err)
	}

	log.Printf("bootstrap id: %s", bootstrap.Host.ID())
	log.Printf("peer1 id: %s", peer1.Host.ID())
	log.Printf("peer2 id: %s", peer2.Host.ID())

	log.Println("Waitting 10s for finding peers testing...")
	time.Sleep(10 * time.Second)
	peer1info, err := peer1.FindPeers(ctx, mockRendezvousString)
	if err != nil {
		t.Errorf("Find peer1 peers err:%s", err)
	}
	peer2info, err := peer2.FindPeers(ctx, mockRendezvousString)
	if err != nil {
		t.Errorf("Find peer2 peers err:%s", err)
	}
	if len(peer2info) == len(peer1info) && len(peer1info) == 2 {
		for _, p2 := range peer2info {
			peerexist := false
			log.Printf("check addr %s", p2)
			for _, p1 := range peer1info {
				if p1.String() == p2.String() {
					peerexist = true
					log.Printf("addr %s exist", p2)
				}
			}
			if peerexist != true {
				t.Errorf("peer2 address not exist in the peer1 %s", p2.String())
			}
		}
	} else {
		t.Errorf("Findpeers peer1 != peer2 %s %s", peer1info, peer2info)
	}

	cancel()

	timer2 := time.NewTimer(time.Second * 30)
	go func() {
		<-timer2.C
		cancel()
	}()

	select {
	case <-ctx.Done():
		log.Println("cancel all nodes after 60s")
		return
	}
}
