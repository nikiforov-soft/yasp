package metrics

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/internal/syncx"
)

type collector struct {
	metricsMapping *config.MetricsMapping
	metrics        *syncx.Map[string, *metric]
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc(
		prometheus.BuildFQName(c.metricsMapping.Namespace, c.metricsMapping.Subsystem, c.metricsMapping.Name),
		c.metricsMapping.Description,
		c.metricsMapping.Labels,
		nil,
	)
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.metrics.Range(func(_ string, m *metric) bool {
		desc := c.buildDesc()
		labels := flattenLabels(c.metricsMapping.Labels, m.labels)
		if metricValue, ok := c.collectMetrics(desc, m, labels); ok {
			ch <- metricValue
		}
		return true
	})
}

func (c *collector) buildDesc() *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(c.metricsMapping.Namespace, c.metricsMapping.Subsystem, c.metricsMapping.Name),
		c.metricsMapping.Description,
		nil,
		nil,
	)
}

func (c *collector) collectMetrics(desc *prometheus.Desc, m *metric, labels []string) (prometheus.Metric, bool) {
	switch strings.ToLower(c.metricsMapping.Type) {
	case "counter":
		return prometheus.MustNewConstMetricWithCreatedTimestamp(
			desc,
			prometheus.CounterValue,
			m.value,
			m.timestamp,
			labels...,
		), true
	case "gauge":
		return prometheus.NewMetricWithTimestamp(m.timestamp, prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			m.value,
			labels...,
		)), true
	case "histogram":
		buckets := make(map[float64]uint64)
		for bucket, count := range m.histogramBuckets {
			buckets[bucket] = uint64(count)
		}

		return prometheus.MustNewConstHistogramWithCreatedTimestamp(
			desc,
			m.histogramCount,
			m.histogramSum,
			buckets,
			m.timestamp,
			labels...,
		), true
	case "summary":
		return prometheus.MustNewConstSummaryWithCreatedTimestamp(
			desc,
			m.summaryCount,
			m.summarySum,
			m.summaryQuantiles,
			m.timestamp,
			labels...,
		), true
	default:
		logrus.WithField("type", c.metricsMapping.Type).Warn("unknown metric type")
		return nil, false
	}
}
