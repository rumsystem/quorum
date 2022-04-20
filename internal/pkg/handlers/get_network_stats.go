package handlers

import (
	"time"

	"github.com/rumsystem/quorum/internal/pkg/stats"
)

func GetNetworkStats(start, end *time.Time) (*stats.NetworkStatsSummary, error) {
	return stats.GetStatsDB().ParseNetworkLog(start, end)
}
