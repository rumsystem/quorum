package scorers_test

import (
	"context"
	"io"
	"math"
	"testing"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p/peerdata"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p/scorers"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(io.Discard)
	m.Run()
}

// roundScore returns score rounded in accordance with the score manager's rounding factor.
func roundScore(score float64) float64 {
	return math.Round(score*scorers.ScoreRoundingFactor) / scorers.ScoreRoundingFactor
}

type testPeerStatuses struct {
	store   *peerdata.Store
	scorers *scorers.Service
}

func newTestPeerStatuses(ctx context.Context, maxPeers int, config *scorers.Config) *testPeerStatuses {
	if config == nil {
		config = &scorers.Config{}
	}
	store := peerdata.NewStore(ctx, &peerdata.StoreConfig{MaxPeers: maxPeers})
	return &testPeerStatuses{
		store:   store,
		scorers: scorers.NewService(ctx, store, config),
	}
}

func (p *testPeerStatuses) Scorers() *scorers.Service {
	return p.scorers
}

func (p *testPeerStatuses) Add(_ interface{}, pid peer.ID, _ interface{}, direction network.Direction) {
	p.store.Lock()
	defer p.store.Unlock()
	data := p.store.PeerDataGetOrCreate(pid)
	data.Direction = direction
}
