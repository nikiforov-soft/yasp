package metrics

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/internal/syncx"
)

type collector struct {
	metricsMapping *config.MetricsMapping
	metrics        *syncx.Map[string, *metric]
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.buildDesc()
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.metrics.Range(func(_ string, m *metric) bool {
		metricValue, err := c.collectMetrics(m)
		if err != nil {
			ch <- prometheus.NewInvalidMetric(c.buildDesc(), err)
		} else {
			ch <- prometheus.NewMetricWithTimestamp(m.timestamp, metricValue)
		}
		return true
	})
}

func (c *collector) buildDesc() *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(c.metricsMapping.Namespace, c.metricsMapping.Subsystem, c.metricsMapping.Name),
		c.metricsMapping.Description,
		c.metricsMapping.Labels,
		nil,
	)
}

func (c *collector) collectMetrics(m *metric) (prometheus.Metric, error) {
	desc := c.buildDesc()
	labels := flattenLabels(c.metricsMapping.Labels, m.labels)

	switch strings.ToLower(c.metricsMapping.Type) {
	case "counter":
		return prometheus.NewConstMetric(
			desc,
			prometheus.CounterValue,
			m.value,
			labels...,
		)
	case "gauge":
		return prometheus.NewConstMetric(
			desc,
			prometheus.GaugeValue,
			m.value,
			labels...,
		)
	case "histogram":
		buckets := make(map[float64]uint64)
		for bucket, count := range m.histogramBuckets {
			buckets[bucket] = uint64(count)
		}
		return prometheus.NewConstHistogram(
			desc,
			m.histogramCount,
			m.histogramSum,
			buckets,
			labels...,
		)
	case "summary":
		return prometheus.NewConstSummary(
			desc,
			m.summaryCount,
			m.summarySum,
			m.summaryQuantiles,
			labels...,
		)
	default:
		return nil, fmt.Errorf("unsupported metric type: %s", c.metricsMapping.Type)
	}
}
