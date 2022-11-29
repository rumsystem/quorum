package p2p

import (
	"context"
	"math"
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
	scorers         *scorers.Service
	store           *peerdata.Store
	rand            *localcrypto.Rand
	rateLimiter     *leakybucket.Collector
	blocksPerSecond uint64
	capacityWeight  float64
}

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

func (rps *RumGroupPeerStore) filterPeers(ctx context.Context, peers []peer.ID, peersPercentage float64) []peer.ID {
	//ctx, span := trace.StartSpan(ctx, "initialsync.filterPeers")
	//defer span.End()

	if len(peers) == 0 {
		return peers
	}
	badscorer := rps.scorers.BadResponsesScorer()
	goodpeers := []peer.ID{}
	for _, peer := range peers {
		isbad := badscorer.IsBadPeer(peer)
		if isbad == false {
			goodpeers = append(goodpeers, peer)
		}
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

	scorer := rps.scorers.BlockProviderScorer()
	peers = scorer.WeightSorted(rps.rand, goodpeers, func(peerID peer.ID, blockProviderScore float64) float64 {
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
