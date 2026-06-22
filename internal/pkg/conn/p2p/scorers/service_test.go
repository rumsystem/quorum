package scorers_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p/scorers"
	"github.com/stretchr/testify/assert"
)

func TestScorers_Service_Init(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	batchSize := uint64(10)

	t.Run("default config", func(t *testing.T) {
		peerStatuses := newTestPeerStatuses(ctx, 30, &scorers.Config{})

		t.Run("bad responses scorer", func(t *testing.T) {
			params := peerStatuses.Scorers().BadResponsesScorer().Params()
			assert.Equal(t, scorers.DefaultBadResponsesThreshold, params.Threshold, "Unexpected threshold value")
			assert.Equal(t, scorers.DefaultBadResponsesDecayInterval, params.DecayInterval, "Unexpected decay interval value")
		})

		t.Run("block providers scorer", func(t *testing.T) {
			params := peerStatuses.Scorers().BlockProviderScorer().Params()
			assert.Equal(t, scorers.DefaultBlockProviderProcessedBatchWeight, params.ProcessedBatchWeight)
			assert.Equal(t, scorers.DefaultBlockProviderProcessedBlocksCap, params.ProcessedBlocksCap)
			assert.Equal(t, scorers.DefaultBlockProviderDecayInterval, params.DecayInterval)
			assert.Equal(t, scorers.DefaultBlockProviderDecay, params.Decay)
			assert.Equal(t, scorers.DefaultBlockProviderStalePeerRefreshInterval, params.StalePeerRefreshInterval)
		})
	})

	t.Run("explicit config", func(t *testing.T) {
		peerStatuses := newTestPeerStatuses(ctx, 30, &scorers.Config{
			BadResponsesScorerConfig: &scorers.BadResponsesScorerConfig{
				Threshold:     2,
				DecayInterval: time.Minute,
			},
			BlockProviderScorerConfig: &scorers.BlockProviderScorerConfig{
				ProcessedBatchWeight:     0.2,
				ProcessedBlocksCap:       batchSize * 5,
				DecayInterval:            time.Minute,
				Decay:                    16,
				StalePeerRefreshInterval: 5 * time.Hour,
			},
		})

		t.Run("bad responses scorer", func(t *testing.T) {
			params := peerStatuses.Scorers().BadResponsesScorer().Params()
			assert.Equal(t, 2, params.Threshold, "Unexpected threshold value")
			assert.Equal(t, time.Minute, params.DecayInterval, "Unexpected decay interval value")
		})

		t.Run("block provider scorer", func(t *testing.T) {
			params := peerStatuses.Scorers().BlockProviderScorer().Params()
			assert.Equal(t, 0.2, params.ProcessedBatchWeight)
			assert.Equal(t, batchSize*5, params.ProcessedBlocksCap)
			assert.Equal(t, time.Minute, params.DecayInterval)
			assert.Equal(t, uint64(16), params.Decay)
			assert.Equal(t, 5*time.Hour, params.StalePeerRefreshInterval)
			assert.Equal(t, 1.0, peerStatuses.Scorers().BlockProviderScorer().MaxScore())
		})
	})
}

func TestScorers_Service_Score(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	batchSize := uint64(10)

	peerScores := func(s *scorers.Service, pids []peer.ID) map[string]float64 {
		scores := make(map[string]float64, len(pids))
		for _, pid := range pids {
			scores[string(pid)] = s.Score(pid)
		}
		return scores
	}

	blkProviderScorers := func(s *scorers.Service, pids []peer.ID) map[string]float64 {
		scores := make(map[string]float64, len(pids))
		for _, pid := range pids {
			scores[string(pid)] = s.BlockProviderScorer().Score(pid)
		}
		return scores
	}

	pack := func(_ *scorers.Service, s1, s2, s3 float64) map[string]float64 {
		return map[string]float64{
			"peer1": roundScore(s1),
			"peer2": roundScore(s2),
			"peer3": roundScore(s3),
		}
	}

	setupScorer := func() (*scorers.Service, []peer.ID) {
		peerStatuses := newTestPeerStatuses(ctx, 30, &scorers.Config{
			BadResponsesScorerConfig:  &scorers.BadResponsesScorerConfig{Threshold: 5},
			BlockProviderScorerConfig: &scorers.BlockProviderScorerConfig{Decay: 64},
		})
		s := peerStatuses.Scorers()
		pids := []peer.ID{"peer1", "peer2", "peer3"}
		for _, pid := range pids {
			peerStatuses.Add(nil, pid, nil, network.DirUnknown)
			assert.Equal(t, float64(0), s.Score(pid), "Unexpected score for not yet used peer")
		}
		return s, pids
	}

	t.Run("no peer registered", func(t *testing.T) {
		peerStatuses := newTestPeerStatuses(ctx, 0, &scorers.Config{})
		s := peerStatuses.Scorers()
		assert.Equal(t, 0.0, s.BadResponsesScorer().Score("peer1"))
		assert.Equal(t, s.BlockProviderScorer().MaxScore(), s.BlockProviderScorer().Score("peer1"))
		assert.Equal(t, 0.0, s.Score("peer1"))
	})

	t.Run("bad responses score", func(t *testing.T) {
		s, pids := setupScorer()
		// Peers start at neutral score; bad responses add weighted penalties.
		startScore := float64(0)
		penalty := (-10 / float64(s.BadResponsesScorer().Params().Threshold)) * 0.5

		// Update peers' stats and test the effect on peer order.
		s.BadResponsesScorer().Increment("peer2")
		assert.Equal(t, pack(s, startScore, startScore+penalty, startScore), peerScores(s, pids))
		s.BadResponsesScorer().Increment("peer1")
		s.BadResponsesScorer().Increment("peer1")
		assert.Equal(t, pack(s, startScore+2*penalty, startScore+penalty, startScore), peerScores(s, pids))

		// See how decaying affects order of peers.
		s.BadResponsesScorer().Decay()
		assert.Equal(t, pack(s, startScore+penalty, startScore, startScore), peerScores(s, pids))
		s.BadResponsesScorer().Decay()
		assert.Equal(t, pack(s, startScore, startScore, startScore), peerScores(s, pids))
	})

	t.Run("block providers score", func(t *testing.T) {
		s, pids := setupScorer()
		s1 := s.BlockProviderScorer()
		startScore := s.BlockProviderScorer().MaxScore()
		batchWeight := s1.Params().ProcessedBatchWeight

		// Partial batch.
		s1.IncrementProcessedBlocks("peer1", batchSize/4)
		assert.Equal(t, 0.0, s.BlockProviderScorer().Score("peer1"), "Unexpected %q score", "peer1")

		// Single batch.
		s1.IncrementProcessedBlocks("peer1", batchSize)
		assert.Equal(t, pack(s, batchWeight, startScore, startScore), blkProviderScorers(s, pids), "Unexpected scores")

		// Multiple batches.
		s1.IncrementProcessedBlocks("peer2", batchSize*4)
		assert.Equal(t, pack(s, batchWeight, batchWeight*4, startScore), blkProviderScorers(s, pids), "Unexpected scores")

		// Partial batch.
		s1.IncrementProcessedBlocks("peer3", batchSize/2)
		assert.Equal(t, pack(s, batchWeight, batchWeight*4, 0), blkProviderScorers(s, pids), "Unexpected scores")

		// See effect of decaying.
		assert.Equal(t, batchSize+batchSize/4, s1.ProcessedBlocks("peer1"))
		assert.Equal(t, batchSize*4, s1.ProcessedBlocks("peer2"))
		assert.Equal(t, batchSize/2, s1.ProcessedBlocks("peer3"))
		assert.Equal(t, pack(s, batchWeight, batchWeight*4, 0), blkProviderScorers(s, pids), "Unexpected scores")
		s1.Decay()
		assert.Equal(t, uint64(0), s1.ProcessedBlocks("peer1"))
		assert.Equal(t, uint64(0), s1.ProcessedBlocks("peer2"))
		assert.Equal(t, uint64(0), s1.ProcessedBlocks("peer3"))
		assert.Equal(t, pack(s, 0, 0, 0), blkProviderScorers(s, pids), "Unexpected scores")
	})

	t.Run("overall score", func(t *testing.T) {
		s, _ := setupScorer()
		s1 := s.BlockProviderScorer()
		s2 := s.BadResponsesScorer()
		penalty := (-10 / float64(s.BadResponsesScorer().Params().Threshold)) * 0.5

		// Full score, no penalty.
		s1.IncrementProcessedBlocks("peer1", batchSize*5)
		assert.Equal(t, float64(0), s.Score("peer1"))
		// Now, adjust score by introducing penalty for bad responses.
		s2.Increment("peer1")
		s2.Increment("peer1")
		assert.Equal(t, roundScore(2*penalty), s.Score("peer1"), "Unexpected overall score")
		// If peer continues to misbehave, score becomes negative.
		s2.Increment("peer1")
		assert.Equal(t, roundScore(3*penalty), s.Score("peer1"), "Unexpected overall score")
	})
}

func TestScorers_Service_loop(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	peerStatuses := newTestPeerStatuses(ctx, 30, &scorers.Config{
		BadResponsesScorerConfig: &scorers.BadResponsesScorerConfig{
			Threshold:     5,
			DecayInterval: 50 * time.Millisecond,
		},
		BlockProviderScorerConfig: &scorers.BlockProviderScorerConfig{
			DecayInterval: 25 * time.Millisecond,
			Decay:         64,
		},
	})
	s1 := peerStatuses.Scorers().BadResponsesScorer()
	s2 := peerStatuses.Scorers().BlockProviderScorer()

	pid1 := peer.ID("peer1")
	peerStatuses.Add(nil, pid1, nil, network.DirUnknown)
	for i := 0; i < s1.Params().Threshold+5; i++ {
		s1.Increment(pid1)
	}
	assert.Equal(t, true, s1.IsBadPeer(pid1), "Peer should be marked as bad")

	s2.IncrementProcessedBlocks("peer1", 221)
	assert.Equal(t, uint64(221), s2.ProcessedBlocks("peer1"))

	done := make(chan struct{}, 1)
	go func() {
		defer func() {
			done <- struct{}{}
		}()
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if !s1.IsBadPeer(pid1) && s2.ProcessedBlocks("peer1") == 0 {
					return
				}
			case <-ctx.Done():
				t.Error("Timed out")
				return
			}
		}
	}()

	<-done
	assert.Equal(t, false, s1.IsBadPeer(pid1), "Peer should not be marked as bad")
	assert.Equal(t, uint64(0), s2.ProcessedBlocks("peer1"), "No blocks are expected")
}

func TestScorers_Service_IsBadPeer(t *testing.T) {
	peerStatuses := newTestPeerStatuses(context.Background(), 30, &scorers.Config{
		BadResponsesScorerConfig: &scorers.BadResponsesScorerConfig{
			Threshold:     2,
			DecayInterval: 50 * time.Second,
		},
	})

	assert.Equal(t, false, peerStatuses.Scorers().IsBadPeer("peer1"))
	peerStatuses.Scorers().BadResponsesScorer().Increment("peer1")
	peerStatuses.Scorers().BadResponsesScorer().Increment("peer1")
	assert.Equal(t, true, peerStatuses.Scorers().IsBadPeer("peer1"))
}

func TestScorers_Service_BadPeers(t *testing.T) {
	peerStatuses := newTestPeerStatuses(context.Background(), 30, &scorers.Config{
		BadResponsesScorerConfig: &scorers.BadResponsesScorerConfig{
			Threshold:     2,
			DecayInterval: 50 * time.Second,
		},
	})

	assert.Equal(t, false, peerStatuses.Scorers().IsBadPeer("peer1"))
	assert.Equal(t, false, peerStatuses.Scorers().IsBadPeer("peer2"))
	assert.Equal(t, false, peerStatuses.Scorers().IsBadPeer("peer3"))
	assert.Equal(t, 0, len(peerStatuses.Scorers().BadPeers()))
	for _, pid := range []peer.ID{"peer1", "peer3"} {
		peerStatuses.Scorers().BadResponsesScorer().Increment(pid)
		peerStatuses.Scorers().BadResponsesScorer().Increment(pid)
	}
	assert.Equal(t, true, peerStatuses.Scorers().IsBadPeer("peer1"))
	assert.Equal(t, false, peerStatuses.Scorers().IsBadPeer("peer2"))
	assert.Equal(t, true, peerStatuses.Scorers().IsBadPeer("peer3"))
	assert.Equal(t, 2, len(peerStatuses.Scorers().BadPeers()))
}
