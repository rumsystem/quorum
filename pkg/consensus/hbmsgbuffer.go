package consensus

import (
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

type HBMsgBuffer struct {
	queueId string
}

func NewHBMsgBuffer(queueId string) *HBMsgBuffer {
	q := &HBMsgBuffer{
		queueId: queueId,
	}
	return q
}

func (h *HBMsgBuffer) AddMsg(msg *quorumpb.HBMsgv1) error {
	return nodectx.GetNodeCtx().GetChainStorage().AddMsgHBB(msg, h.queueId)
}

func (h *HBMsgBuffer) GetAllMsg() ([]*quorumpb.HBMsgv1, error) {
	msgs, err := nodectx.GetNodeCtx().GetChainStorage().GetAllMsgHBB(h.queueId)
	if err != nil {
		return nil, err
	}

	return msgs, nil
}

func (h *HBMsgBuffer) GetBufferLen() (int, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GeBufferedMsgLenHBB(h.queueId)
}

func (h *HBMsgBuffer) GetMsgsByEpoch(epoch uint64) ([]*quorumpb.HBMsgv1, error) {
	return nodectx.GetNodeCtx().GetChainStorage().GetMsgsByEpochHBB(h.queueId, epoch)
}

func (h *HBMsgBuffer) DelMsgById(epoch uint64, msgId string) error {
	return nodectx.GetNodeCtx().GetChainStorage().RemoveMsgByMsgId(h.queueId, epoch, msgId)
}

func (h *HBMsgBuffer) DelMsgsByEpoch(epoch uint64) error {
	return nodectx.GetNodeCtx().GetChainStorage().RemoveMsgByEpochHBB(h.queueId, epoch)
}

func (h *HBMsgBuffer) ClearBuffer() error {
	return nodectx.GetNodeCtx().GetChainStorage().RemoveAllMsgHBB(h.queueId)
}
