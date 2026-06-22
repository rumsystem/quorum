package scorers_test

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p/scorers"
	"github.com/stretchr/testify/assert"
)

func TestScorers_PeerStatus_Score(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peerStatuses := newTestPeerStatuses(ctx, 30, &scorers.Config{})
	scorer := peerStatuses.Scorers().PeerStatusScorer()

	assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score for nonexistent peer")

	scorer.SetPeerStatus("peer1", nil)
	assert.Equal(t, 0.0, scorer.Score("peer1"), "Unexpected score for valid peer")

	scorer.SetPeerStatus("peer2", scorers.ErrWrongForkDigestVersion)
	assert.Equal(t, scorers.BadPeerScore, scorer.Score("peer2"), "Unexpected score for bad peer")
}

func TestScorers_PeerStatus_BadPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peerStatuses := newTestPeerStatuses(ctx, 30, &scorers.Config{})
	scorer := peerStatuses.Scorers().PeerStatusScorer()

	scorer.SetPeerStatus(peer.ID("peer1"), nil)
	scorer.SetPeerStatus(peer.ID("peer2"), scorers.ErrInvalidRequest)
	scorer.SetPeerStatus(peer.ID("peer3"), nil)

	assert.Equal(t, false, scorer.IsBadPeer("peer1"))
	assert.Equal(t, true, scorer.IsBadPeer("peer2"))
	assert.Equal(t, false, scorer.IsBadPeer("peer3"))
	assert.Equal(t, []peer.ID{peer.ID("peer2")}, scorer.BadPeers())
}
