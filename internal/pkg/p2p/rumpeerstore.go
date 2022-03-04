package p2p

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"math/rand"
	"sync"
	"time"
)

type RumPeer struct {
	Id      peer.ID
	TTL     time.Duration
	Updated time.Time
}

//groupid to RumPeer
type RumGroupPeerStore struct {
	store sync.Map
}

var ignoregroupid string = "00000000-0000-0000-0000-000000000000"

func (rps *RumGroupPeerStore) Save(groupid string, Id peer.ID, TTL time.Duration) {
	if groupid == ignoregroupid {
		return
	}
	list, succ := rps.store.Load(groupid)
	if succ == false {
		newpeerlist := &[]RumPeer{}
		rps.store.Store(groupid, newpeerlist)
		list, succ = rps.store.Load(groupid)
	}

	newrumpeer := RumPeer{Id: Id, TTL: TTL, Updated: time.Now()}
	peerlist := *list.(*[]RumPeer)
	//if peer exist? update it
	for i, p := range peerlist {
		if p.Id == newrumpeer.Id {
			peerlist[i] = newrumpeer
			rps.store.Store(groupid, &peerlist) //update the sync.Map
			return
		}
	}
	peerlist = append(peerlist, newrumpeer)
	rps.store.Store(groupid, &peerlist) //update the sync.Map
}

func (rps *RumGroupPeerStore) AddIgnorePeer(Id peer.ID) {
	rps.Save(ignoregroupid, Id, time.Duration(5*time.Hour))
}

func (rps *RumGroupPeerStore) Get(groupid string) []peer.ID {
	list, succ := rps.store.Load(groupid)
	result := []peer.ID{}
	if succ == false {
		return result
	}
	peerlist := *list.(*[]RumPeer)
	for _, p := range peerlist {
		if time.Now().Sub(p.Updated) <= p.TTL {
			result = append(result, p.Id)
		}
	}
	return result
}

func (rps *RumGroupPeerStore) GetRandomPeer(groupid string, count int, connectPeers []peer.ID) []peer.ID {
	savedpeers := rps.Get(groupid)
	if len(savedpeers) == count {
		return savedpeers
	} else if len(savedpeers) > count {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(savedpeers), func(i, j int) { savedpeers[i], savedpeers[j] = savedpeers[j], savedpeers[i] })
		return savedpeers[0:count]
	} else {
		ignoredpeers := rps.Get(groupid)
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(connectPeers), func(i, j int) { connectPeers[i], connectPeers[j] = connectPeers[j], connectPeers[i] })

		for _, newp := range connectPeers {
			for _, ip := range ignoredpeers {
				if ip == newp {
					break
				}
			}
			for _, sp := range savedpeers {
				if sp == newp {
					break
				}
			}
			savedpeers = append(savedpeers, newp)
			if len(savedpeers) == count {
				return savedpeers
			}
		}
		return savedpeers

	}
	return nil
}
