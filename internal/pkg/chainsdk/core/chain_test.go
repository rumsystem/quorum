package chain

import (
	"testing"
)

func TestGroups(t *testing.T) {
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// mockRendezvousString := "testrendezvousuniqstring"
	// bootstrap, peer1, peer2, _, peer1keys, peer2keys, err := testnode.Run2nodes(ctx, mockRendezvousString)
	// if err != nil {
	// 	t.Errorf("create test peers err:%s", err)
	// }

	// log.Printf("bootstrap id: %s", bootstrap.Host.ID())
	// log.Printf("peer1 id: %s", peer1.Host.ID())
	// log.Printf("peer2 id: %s", peer2.Host.ID())

	// log.Println("Waitting 10s for finding peers testing...")
	// time.Sleep(10 * time.Second)

	// fmt.Println(peer2keys)
	// datapath, err := ioutil.TempDir("", "peer1")
	// log.Printf("chain node 1 setup at: %s", datapath)
	// chain.InitCtx(ctx, datapath)
	// chain.GetChainCtx().Privatekey = peer1keys.PrivKey
	// chain.GetChainCtx().PublicKey = peer1keys.PubKey
	// //TODO:
	// peer1id, err := peer.IDFromPublicKey(peer1keys.PubKey)
	// chain.GetChainCtx().PeerId = peer1id

	// cancel()

	// timer2 := time.NewTimer(time.Second * 60)
	// go func() {
	// 	<-timer2.C
	// 	cancel()
	// }()

	// select {
	// case <-ctx.Done():
	// 	log.Println("cancel all nodes after 60s")
	// 	return
	// }
}
