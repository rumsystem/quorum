package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "quorum"
)

var (
	ActionType = struct {
		ConnectPeer      string
		JoinTopic        string
		SubscribeTopic   string
		PublishToTopic   string
		PublishToPeerid  string
		PublishToStream  string
		ReceiveFromTopic string
		RumChainData     string
		RumRelayReq      string
		RumRelayResp     string
	}{
		ConnectPeer:      "connect_peer",
		JoinTopic:        "join_topic",
		SubscribeTopic:   "subscribe_topic",
		PublishToTopic:   "publish_to_topic",
		PublishToPeerid:  "publish_to_peerid",
		PublishToStream:  "publish_to_stream",
		ReceiveFromTopic: "receive_from_topic",
		RumChainData:     "rum_chain_data",
		RumRelayReq:      "rum_relay_req",
		RumRelayResp:     "rum_relay_resp",
	}

	SuccessCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "success_total",
			Help:      "The total number of successful action",
		},
		[]string{"action"},
	)

	FailedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "failed_total",
			Help:      "The total number of failed action",
		},
		[]string{"action"},
	)

	InBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "in_bytes",
			Help:      "Current count of bytes received",
		},
		[]string{"action"},
	)

	InBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "in_bytes_total",
			Help:      "Total count of bytes received",
		},
		[]string{"action"},
	)

	OutBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "out_bytes",
			Help:      "Current count of bytes sent",
		},
		[]string{"action"},
	)

	OutBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "out_bytes_total",
			Help:      "Total count of bytes sent",
		},
		[]string{"action"},
	)
)
