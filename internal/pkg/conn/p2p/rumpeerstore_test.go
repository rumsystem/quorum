package p2p

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p/scorers"
)

func TestRumGroupPeerStoreFilterPeersDropsBadPeers(t *testing.T) {
	rgp := NewRumGroupPeerStore()
	pids := []peer.ID{"peer1", "peer2", "peer3", "peer4"}

	for i := 0; i < scorers.DefaultBadResponsesThreshold; i++ {
		rgp.Scorers().BadResponsesScorer().Increment(pids[1])
	}

	filtered := rgp.filterPeers(context.Background(), pids, 1.0)
	for _, pid := range filtered {
		if pid == pids[1] {
			t.Fatalf("bad peer %s was not filtered: %v", pid, filtered)
		}
	}
	if len(filtered) != 3 {
		t.Fatalf("expected three good peers, got %d: %v", len(filtered), filtered)
	}
}

func TestTrimPeersKeepsMinimumAndCapsAtAvailablePeers(t *testing.T) {
	pids := []peer.ID{"peer1", "peer2", "peer3", "peer4", "peer5"}

	trimmed := trimPeers(pids, 0.2)
	if len(trimmed) != 3 {
		t.Fatalf("expected minimum of three peers, got %d: %v", len(trimmed), trimmed)
	}

	trimmed = trimPeers(pids[:2], 0.1)
	if len(trimmed) != 2 {
		t.Fatalf("expected cap at available peers, got %d: %v", len(trimmed), trimmed)
	}
}
