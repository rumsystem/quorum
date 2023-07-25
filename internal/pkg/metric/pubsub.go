package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	pubsubNS = "pubsub"
)

var (
	PubSubActionType = struct {
		ConnectPeer      string
		JoinTopic        string
		SubscribeTopic   string
		PublishToTopic   string
		ReceiveFromTopic string
	}{
		ConnectPeer:      "connect_peer",
		JoinTopic:        "join_topic",
		SubscribeTopic:   "subscribe_topic",
		PublishToTopic:   "publish_to_topic",
		ReceiveFromTopic: "receive_from_topic",
	}

	PubSubSuccessCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pubsubNS,
			Name:      "success_total",
			Help:      "The total number of successful action",
		},
		[]string{"action"},
	)

	PubSubFailedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pubsubNS,
			Name:      "failed_total",
			Help:      "The total number of failed action",
		},
		[]string{"action"},
	)

	PubSubInBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: pubsubNS,
			Name:      "in_bytes",
			Help:      "Current count of bytes received",
		},
		[]string{"action"},
	)

	PubSubInBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pubsubNS,
			Name:      "in_bytes_total",
			Help:      "Total count of bytes received",
		},
		[]string{"action"},
	)

	PubSubOutBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: pubsubNS,
			Name:      "out_bytes",
			Help:      "Current count of bytes sent",
		},
		[]string{"action"},
	)

	PubSubOutBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: pubsubNS,
			Name:      "out_bytes_total",
			Help:      "Total count of bytes sent",
		},
		[]string{"action"},
	)
)
