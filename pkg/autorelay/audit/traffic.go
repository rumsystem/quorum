package audit

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/rumsystem/quorum/internal/pkg/storage"
)

/* to record traffic consumption of a peer */

type QuorumTrafficAudit struct {
	db storage.QuorumStorage
}

func NewQuorumTrafficAudit(db storage.QuorumStorage) *QuorumTrafficAudit {
	a := QuorumTrafficAudit{db}
	return &a
}

func (a *QuorumTrafficAudit) OnRelay(src peer.ID, dest peer.ID, count int64) {
	fmt.Printf("%s -> %s: %d bytes", src.String(), dest.String(), count)
}
