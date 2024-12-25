package metrics

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikiforov-soft/yasp/config"
)

type metric struct {
	value     float64
	labels    prometheus.Labels
	timestamp time.Time

	// histogram
	histogramBuckets map[float64]float64
	histogramSum     float64
	histogramCount   uint64

	// summary
	summaryQuantiles map[float64]float64
	summarySum       float64
	summaryCount     uint64
}

func newMetric(mapping *config.MetricsMapping, value float64, labels prometheus.Labels) *metric {
	now := time.Now()

	switch strings.ToLower(mapping.Type) {
	case "histogram":
		buckets := make(map[float64]float64)
		for _, bucket := range mapping.Buckets {
			if value <= bucket {
				buckets[bucket]++
			}
		}
		return &metric{
			histogramBuckets: buckets,
			histogramSum:     value,
			histogramCount:   1,
			labels:           labels,
			timestamp:        now,
		}
	case "summary":
		quantiles := make(map[float64]float64)
		for quantile := range mapping.Objectives {
			quantiles[quantile] = value
		}
		return &metric{
			summaryQuantiles: quantiles,
			summarySum:       value,
			summaryCount:     1,
			labels:           labels,
			timestamp:        now,
		}
	default:
		return &metric{
			value:     value,
			labels:    labels,
			timestamp: now,
		}
	}
}
