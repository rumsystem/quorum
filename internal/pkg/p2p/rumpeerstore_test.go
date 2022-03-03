package p2p

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"io"
	"testing"
	"time"
)

func TestRumGroupPeerStoreUpdate(t *testing.T) {
	//testing add, update, get
	rgp := &RumGroupPeerStore{}
	groupid := "e3326fcf-0df4-4388-9355-48b184c5a3ce"
	peerid1, _ := peer.Decode("16Uiu2HAmCxKwe3h1MiQmgrWsuDpsdRXz1Tr12iuUJ8iEjoCpi7BY")
	ttl := time.Duration(20 * time.Minute)
	rgp.Save(groupid, peerid1, ttl)

	peerid2, _ := peer.Decode("16Uiu2HAm17k6DX4ZkDPYw1H915MxZ4K11qqBvkueFhgdcHRWkX4G")
	rgp.Save(groupid, peerid2, ttl)

	ttl2 := time.Duration(30 * time.Minute)
	rgp.Save(groupid, peerid2, ttl2)

	peerlist := rgp.Get(groupid)

	if len(peerlist) != 2 {
		t.Errorf("RumGroupPeerStore length %d error expect: %d", len(peerlist), 2)
	}

	groupid1 := "f6a92871-1df5-4e23-b9c6-95745bb4236d"
	rgp.Save(groupid1, peerid2, ttl)
	peerlist2 := rgp.Get(groupid1)
	if len(peerlist2) != 1 {
		t.Errorf("RumGroupPeerStore length %d error expect: %d", len(peerlist2), 1)
	}

}

func TestRumGroupPeerStoreTTL(t *testing.T) {
	rgp := &RumGroupPeerStore{}
	groupid1 := "4455c41b-4098-4be1-ba53-a2c4293ca1b5"
	peerid1, _ := peer.Decode("16Uiu2HAmCxKwe3h1MiQmgrWsuDpsdRXz1Tr12iuUJ8iEjoCpi7BY")
	veryshortttl := time.Duration(2 * time.Second)
	rgp.Save(groupid1, peerid1, veryshortttl)
	peerlist1 := rgp.Get(groupid1)
	if len(peerlist1) != 1 {
		t.Errorf("RumGroupPeerStore length %d error expect: %d", len(peerlist1), 1)
	}
	time.Sleep(3 * time.Second)
	t.Log("wait for ttl expired...")
	peerlist1 = rgp.Get(groupid1)
	if len(peerlist1) != 0 {
		t.Errorf("RumGroupPeerStore length %d error expect: %d", len(peerlist1), 0)
	}
}

func TestRumGroupPeerStoreRandom(t *testing.T) {
	rgp := &RumGroupPeerStore{}
	groupid1 := "4455c41b-4098-4be1-ba53-a2c4293ca1b5"
	ttl := time.Duration(20 * time.Minute)
	addpeernum := 10
	connectpeernum := 20
	samplenum := 3

	var r io.Reader
	r = rand.Reader
	for i := 0; i < addpeernum; i++ {
		priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
		newId, _ := peer.IDFromPrivateKey(priv)
		rgp.Save(groupid1, newId, ttl)
	}

	connectpeers := []peer.ID{}
	for i := 0; i < connectpeernum; i++ {
		priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
		newId, _ := peer.IDFromPrivateKey(priv)
		connectpeers = append(connectpeers, newId)
	}

	allpeers := rgp.Get(groupid1)

	randomlist1 := rgp.GetRandomPeer(groupid1, samplenum, []peer.ID{})
	randomlist2 := rgp.GetRandomPeer(groupid1, samplenum, []peer.ID{})

	for i := 0; i < samplenum; i++ {
		peer1included := false
		peer2included := false
		for j := 0; j < addpeernum; j++ {
			if randomlist1[i] == allpeers[j] {
				peer1included = true
			}
			if randomlist2[i] == allpeers[j] {
				peer2included = true
			}
		}
		if peer1included != true || peer2included != true {
			t.Errorf("random peer list not included in all peers.")
		}
	}

	randomlist3 := rgp.GetRandomPeer(groupid1, addpeernum+samplenum, connectpeers)
	if len(randomlist3) != addpeernum+samplenum {
		t.Errorf("RumGroupPeerStore length %d error expect: %d", len(randomlist3), addpeernum+samplenum)
	}

}

func TestRumGroupPeerStoreRandomWithoutPreSaved(t *testing.T) {
	rgp := &RumGroupPeerStore{}

	groupid1 := "4455c41b-4098-4be1-ba53-a2c4293ca1b5"
	connectpeernum := 20
	samplenum := 3

	var r io.Reader
	r = rand.Reader

	connectpeers := []peer.ID{}
	for i := 0; i < connectpeernum; i++ {
		priv, _, _ := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
		newId, _ := peer.IDFromPrivateKey(priv)
		connectpeers = append(connectpeers, newId)
	}

	randomlist1 := rgp.GetRandomPeer(groupid1, samplenum, connectpeers)

	if len(randomlist1) != samplenum {
		t.Errorf("RumGroupPeerStore length %d error expect: %d", len(randomlist1), samplenum)
	}

}
