package consensus

import (
	"math/rand"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

// just a simple wrap of HBB Trx Buffer DB
type TrxBuffer struct {
	queueId string
}

func NewTrxBuffer(queueId string) *TrxBuffer {
	b := &TrxBuffer{
		queueId: queueId,
	}
	rand.Seed(time.Now().UnixNano())
	return b
}

func (b *TrxBuffer) GetBufferLen() (int, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GeBufferedTrxLenHBB(b.queueId)
}

func (b *TrxBuffer) Push(trx *quorumpb.Trx) error {
	return nodectx.GetNodeCtx().GetChainStorage().AddTrxHBB(trx, b.queueId)
}

func (b *TrxBuffer) Delete(trxId string) error {
	return nodectx.GetNodeCtx().GetChainStorage().RemoveTrxHBB(trxId, b.queueId)
}

func (b *TrxBuffer) Clear() error {
	return nodectx.GetNodeCtx().GetChainStorage().RemoveAllTrxHBB(b.queueId)
}

func (b *TrxBuffer) GetTrxById(trxId string) (*quorumpb.Trx, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GetTrxByIdHBB(trxId, b.queueId)
}

// since trx is buffered in *random" way, no sequence is created
// just return the first n items in the slice is enough
// caller should check the length of return trx slice
func (b *TrxBuffer) GetNRandTrx(n int) ([]*quorumpb.Trx, error) {
	//get len
	len, err := nodectx.GetNodeCtx().GetChainStorage().GeBufferedTrxLenHBB(b.queueId)
	if err != nil {
		return nil, err
	}

	trxs, err := nodectx.GetNodeCtx().GetChainStorage().GetAllTrxHBB(b.queueId)

	if n >= len {
		//return all trxs in buffer
		return trxs, err
	} else {
		//return first n trxs in buffer
		return trxs[:n], err
	}
}
