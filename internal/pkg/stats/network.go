package stats

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

var logger = logging.Logger("stats")

const (
	layout   = "20060102150405.999Z0700" // year month day hour minute second ns
	dbKeySep = ":"
)

type NetworkAction string

const (
	// network action for stats
	UnKnownNetworkAction NetworkAction = "unknown_network_action"

	ConnectPeer NetworkAction = "connect_peer"

	// rumexchange
	PublishToStream    NetworkAction = "publish_to_stream"
	PublishToPeerID    NetworkAction = "publish_to_peerid"
	RumRelayReq        NetworkAction = "rum_relay_req"
	RumRelayResp       NetworkAction = "rum_relay_resp"
	RumSessionIfConn   NetworkAction = "rum_session_if_conn"
	RumSessionConnResp NetworkAction = "rum_session_conn_resp"
	RumChainData       NetworkAction = "rum_chain_data"

	// pubsub
	JoinTopic        NetworkAction = "join_topic"
	SubscribeTopic   NetworkAction = "subscribe_topic"
	PublishToTopic   NetworkAction = "publish_to_topic"
	ReceiveFromTopic NetworkAction = "receive_from_topic"
)

func (na NetworkAction) GetByRumMsgType(msgType quorumpb.RumMsgType) NetworkAction {
	switch msgType {
	case quorumpb.RumMsgType_RELAY_REQ:
		return RumRelayReq
	case quorumpb.RumMsgType_RELAY_RESP:
		return RumRelayResp
	case quorumpb.RumMsgType_IF_CONN:
		return RumSessionIfConn
	case quorumpb.RumMsgType_CONN_RESP:
		return RumSessionConnResp
	case quorumpb.RumMsgType_CHAIN_DATA:
		return RumChainData
	default:
		logger.Errorf("unknown msgType: %s", msgType)
		return UnKnownNetworkAction
	}
}

const (
	// StatsKeyPrefix prefix for stats key
	StatsKeyPrefix = "stats"
)

type NetworkStats struct {
	From      string        `json:"from"`
	To        string        `json:"to"`
	Topic     string        `json:"topic"`
	Direction string        `json:"direction"`
	Action    NetworkAction `json:"action"`
	Size      int           `json:"size"` // byte
	Success   bool          `json:"success"`
	CreatedAt *time.Time    `json:"created_at"`
}

func (stats *NetworkStats) ToNetworkStatsSummaryItem() *NetworkStatsSummaryItem {
	var successCount int64
	var failedCount int64
	if stats.Success {
		successCount = 1
	} else {
		failedCount = 1
	}

	var inSize int
	var outSize int
	if stats.Direction == "in" {
		inSize = stats.Size
	} else if stats.Direction == "out" {
		outSize = stats.Size
	}

	return &NetworkStatsSummaryItem{
		Action:       stats.Action,
		SuccessCount: successCount,
		FailedCount:  failedCount,
		InSize:       inSize,
		OutSize:      outSize,
	}
}

type NetworkDBKey struct {
	Prefix   string    `json:"prefix"`
	Datetime time.Time `json:"datetime"`
	Action   string    `json:"action"`
}

func (n *NetworkDBKey) String() string {
	now := n.Datetime.Format(layout)
	prefix := n.Prefix
	if prefix == "" {
		prefix = StatsKeyPrefix
	}
	parts := []string{prefix, now, n.Action}
	return strings.Join(parts, dbKeySep)
}

func ParseDBKey(key string) (*NetworkDBKey, error) {
	parts := strings.Split(key, dbKeySep)
	if len(parts) != 3 {
		err := errors.New("parse db key failed, len(%+v) != 3")
		logger.Error(err.Error())
		return nil, err
	}
	datetime, err := time.Parse(layout, parts[1])
	if err != nil {
		return nil, err
	}
	res := NetworkDBKey{
		Prefix:   parts[0],
		Datetime: datetime,
		Action:   parts[2],
	}
	return &res, nil
}

// GetDBKey returns db key
func (n *NetworkStats) GetDBKey() string {
	key := NetworkDBKey{
		Prefix:   StatsKeyPrefix,
		Datetime: *n.CreatedAt,
		Action:   string(n.Action),
	}
	return key.String()
}

func GetDBKeyPrefixByStr(s string) string {
	parts := []string{StatsKeyPrefix, s}
	return strings.Join(parts, dbKeySep)
}

type NetworkStatsSummaryItem struct {
	Action       NetworkAction `json:"action"`
	SuccessCount int64         `json:"success_count"`
	FailedCount  int64         `json:"failed_count"`
	InSize       int           `json:"in_size"`
	OutSize      int           `json:"out_size"`
}
type NetworkStatsSummary struct {
	Summary map[NetworkAction]*NetworkStatsSummaryItem `json:"summary"`

	sync.Mutex `json:"-"`
}

func NewNetworkStatsSummary() *NetworkStatsSummary {
	summary := NetworkStatsSummary{}
	summary.Summary = make(map[NetworkAction]*NetworkStatsSummaryItem)
	return &summary
}

func (summary *NetworkStatsSummary) Add(n NetworkStatsSummaryItem) {
	summary.Lock()
	defer summary.Unlock()

	item, ok := summary.Summary[n.Action]
	if !ok {
		summary.Summary[n.Action] = &n
	} else {
		summary.Summary[n.Action] = &NetworkStatsSummaryItem{
			Action:       item.Action,
			SuccessCount: item.SuccessCount + n.SuccessCount,
			FailedCount:  item.FailedCount + n.FailedCount,
			InSize:       item.InSize + n.InSize,
			OutSize:      item.OutSize + n.OutSize,
		}
	}
}
