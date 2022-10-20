package p2p

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p/peerdata"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p/scorers"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
)

type RumPeer struct {
	Id      peer.ID
	TTL     time.Duration
	Updated time.Time
}

//groupid to RumPeer
type RumGroupPeerStore struct {
	syncmapstore    sync.Map
	scorers         *scorers.Service
	store           *peerdata.Store
	rand            *localcrypto.Rand
	rateLimiter     *leakybucket.Collector
	blocksPerSecond uint64
	capacityWeight  float64
}

var ignoregroupid string = "00000000-0000-0000-0000-000000000000"

func NewRumGroupPeerStore() *RumGroupPeerStore {
	ctx := context.Background()
	store := peerdata.NewStore(ctx, &peerdata.StoreConfig{
		//MaxPeers: maxLimitBuffer + config.PeerLimit,
		MaxPeers: 20,
	})

	blocksPerSecond := 64
	BlockBatchLimitBurstFactor := 10

	scorerParams := &scorers.Config{}

	myscorers := scorers.NewService(ctx, store, scorerParams)

	rd := localcrypto.NewGenerator()

	allowedBlocksBurst := BlockBatchLimitBurstFactor * blocksPerSecond

	rateLimiter := leakybucket.NewCollector(
		float64(blocksPerSecond), int64(allowedBlocksBurst-blocksPerSecond),
		false /* deleteEmptyBuckets */)

	rps := &RumGroupPeerStore{store: store, scorers: myscorers, rand: rd, rateLimiter: rateLimiter, blocksPerSecond: uint64(blocksPerSecond), capacityWeight: 0.2}
	return rps
}

func (rps *RumGroupPeerStore) Scorers() *scorers.Service {
	return rps.scorers
}

func (rps *RumGroupPeerStore) Save(groupid string, Id peer.ID, TTL time.Duration) {
	//if groupid == ignoregroupid {
	//	return
	//}
	//list, succ := rps.store.Load(groupid)
	//if succ == false {
	//	newpeerlist := &[]RumPeer{}
	//	rps.store.Store(groupid, newpeerlist)
	//	list, succ = rps.store.Load(groupid)
	//}

	//newrumpeer := RumPeer{Id: Id, TTL: TTL, Updated: time.Now()}
	//peerlist := *list.(*[]RumPeer)
	////if peer exist? update it
	//for i, p := range peerlist {
	//	if p.Id == newrumpeer.Id {
	//		peerlist[i] = newrumpeer
	//		rps.store.Store(groupid, &peerlist) //update the sync.Map
	//		return
	//	}
	//}
	//peerlist = append(peerlist, newrumpeer)
	//rps.store.Store(groupid, &peerlist) //update the sync.Map
}

func (rps *RumGroupPeerStore) AddIgnorePeer(Id peer.ID) {
	//	rps.Save(ignoregroupid, Id, time.Duration(5*time.Hour))
}

func (rps *RumGroupPeerStore) Get(groupid string) []peer.ID {
	//list, succ := rps.store.Load(groupid)
	result := []peer.ID{}
	//if succ == false {
	//	return result
	//}
	//peerlist := *list.(*[]RumPeer)
	//for _, p := range peerlist {
	//	if time.Now().Sub(p.Updated) <= p.TTL {
	//		result = append(result, p.Id)
	//	}
	//}
	return result
}

func (rps *RumGroupPeerStore) GetOneRandomPeer(connectPeers []peer.ID) (peer.ID, error) {
	//ignoredpeers := rps.Get(ignoregroupid)
	//rand.Seed(time.Now().UnixNano())
	//rand.Shuffle(len(connectPeers), func(i, j int) { connectPeers[i], connectPeers[j] = connectPeers[j], connectPeers[i] })

	//for _, newp := range connectPeers {
	//	for _, ip := range ignoredpeers {
	//		if ip == newp {
	//			break
	//		}
	//	}
	//	return newp, nil
	//}

	return "", fmt.Errorf("no available peer")
}

func (rps *RumGroupPeerStore) GetRandomPeer(groupid string, count int, connectPeers []peer.ID) []peer.ID {
	//savedpeers := rps.Get(groupid)
	//if len(savedpeers) == count {
	//	return savedpeers
	//} else if len(savedpeers) > count {
	//	rand.Seed(time.Now().UnixNano())
	//	rand.Shuffle(len(savedpeers), func(i, j int) { savedpeers[i], savedpeers[j] = savedpeers[j], savedpeers[i] })
	//	return savedpeers[0:count]
	//} else {
	//	ignoredpeers := rps.Get(groupid)
	//	rand.Seed(time.Now().UnixNano())
	//	rand.Shuffle(len(connectPeers), func(i, j int) { connectPeers[i], connectPeers[j] = connectPeers[j], connectPeers[i] })

	//	for _, newp := range connectPeers {
	//		for _, ip := range ignoredpeers {
	//			if ip == newp {
	//				break
	//			}
	//		}
	//		for _, sp := range savedpeers {
	//			if sp == newp {
	//				break
	//			}
	//		}
	//		savedpeers = append(savedpeers, newp)
	//		if len(savedpeers) == count {
	//			return savedpeers
	//		}
	//	}
	//	return savedpeers

	//}
	return nil
}

func (rps *RumGroupPeerStore) filterPeers(ctx context.Context, peers []peer.ID, peersPercentage float64) []peer.ID {
	//ctx, span := trace.StartSpan(ctx, "initialsync.filterPeers")
	//defer span.End()

	if len(peers) == 0 {
		return peers
	}

	// Sort peers using both block provider score and, custom, capacity based score (see
	// peerFilterCapacityWeight if you want to give different weights to provider's and capacity
	// scores).
	// Scores produced are used as weights, so peers are ordered probabilistically i.e. peer with
	// a higher score has higher chance to end up higher in the list.
	//scorer := f.p2p.Peers().Scorers().BlockProviderScorer()

	//store := peerdata.NewStore(ctx, &peerdata.StoreConfig{
	//	MaxPeers: maxLimitBuffer + config.PeerLimit,
	//})

	//scorer := scorers.Scorers().BlockProviderScorer()
	scorer := rps.scorers.BlockProviderScorer()
	peers = scorer.WeightSorted(rps.rand, peers, func(peerID peer.ID, blockProviderScore float64) float64 {
		remaining, capacity := float64(rps.rateLimiter.Remaining(peerID.String())), float64(rps.rateLimiter.Capacity())
		// When capacity is close to exhaustion, allow less performant peer to take a chance.
		// Otherwise, there's a good chance system will be forced to wait for rate limiter.
		if remaining < float64(rps.blocksPerSecond) {
			return 0.0
		}
		capScore := remaining / capacity
		overallScore := blockProviderScore*(1.0-rps.capacityWeight) + capScore*rps.capacityWeight
		return math.Round(overallScore*scorers.ScoreRoundingFactor) / scorers.ScoreRoundingFactor
	})

	return trimPeers(peers, peersPercentage)
}

func trimPeers(peers []peer.ID, peersPercentage float64) []peer.ID {
	//TODO read value from config
	required := 3
	//required := params.BeaconConfig().MaxPeersToSync
	//if flags.Get().MinimumSyncPeers < required {
	//	required = flags.Get().MinimumSyncPeers
	//}
	// Weak/slow peers will be pushed down the list and trimmed since only percentage of peers is selected.
	limit := uint64(math.Round(float64(len(peers)) * peersPercentage))
	// Limit cannot be less that minimum peers required by sync mechanism.
	limit = utils.Max(limit, uint64(required))
	// Limit cannot be higher than number of peers available (safe-guard).
	limit = utils.Min(limit, uint64(len(peers)))
	return peers[:limit]
}
