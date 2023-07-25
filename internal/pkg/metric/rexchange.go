package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	RexNS = "rumexchange"
)

var (
	RexActionType = struct {
		PublishToStream string
		PublishToPeerid string
		RumChainData    string
		RumRelayReq     string
		RumRelayResp    string
	}{
		PublishToStream: "publish_to_stream",
		PublishToPeerid: "publish_to_peerid",
		RumChainData:    "rum_chain_data",
		RumRelayReq:     "rum_relay_req",
		RumRelayResp:    "rum_relay_resp",
	}

	RexSuccessCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: RexNS,
			Name:      "success_total",
			Help:      "The total number of successful action",
		},
		[]string{"action"},
	)

	RexFailedCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: RexNS,
			Name:      "failed_total",
			Help:      "The total number of failed action",
		},
		[]string{"action"},
	)

	RexInBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: RexNS,
			Name:      "in_bytes",
			Help:      "Current count of bytes received",
		},
		[]string{"action"},
	)

	RexInBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: RexNS,
			Name:      "in_bytes_total",
			Help:      "Total count of bytes received",
		},
		[]string{"action"},
	)

	RexOutBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: RexNS,
			Name:      "out_bytes",
			Help:      "Current count of bytes sent",
		},
		[]string{"action"},
	)

	RexOutBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: RexNS,
			Name:      "out_bytes_total",
			Help:      "Total count of bytes sent",
		},
		[]string{"action"},
	)
)
