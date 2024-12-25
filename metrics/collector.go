package metrics

import (
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
)

type collector struct {
	mapping *config.MetricsMapping
	metrics *sync.Map
}

func (cc *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc(
		prometheus.BuildFQName(cc.mapping.Namespace, cc.mapping.Subsystem, cc.mapping.Name),
		cc.mapping.Description,
		cc.mapping.Labels,
		nil,
	)
}

func (cc *collector) Collect(ch chan<- prometheus.Metric) {
	cc.metrics.Range(func(_, value any) bool {
		m := value.(*metric)
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(cc.mapping.Namespace, cc.mapping.Subsystem, cc.mapping.Name),
			cc.mapping.Description,
			nil,
			nil,
		)

		labels := flattenLabels(cc.mapping.Labels, m.labels)
		switch strings.ToLower(cc.mapping.Type) {
		case "counter":
			ch <- prometheus.MustNewConstMetricWithCreatedTimestamp(
				desc,
				prometheus.CounterValue,
				m.value,
				m.timestamp,
				labels...,
			)
		case "gauge":
			ch <- prometheus.NewMetricWithTimestamp(m.timestamp, prometheus.MustNewConstMetric(
				desc,
				prometheus.GaugeValue,
				m.value,
				labels...,
			))
		case "histogram":
			buckets := make(map[float64]uint64)
			for bucket, count := range m.histogramBuckets {
				buckets[bucket] = uint64(count)
			}

			ch <- prometheus.MustNewConstHistogramWithCreatedTimestamp(
				desc,
				m.histogramCount,
				m.histogramSum,
				buckets,
				m.timestamp,
				labels...,
			)
		case "summary":
			ch <- prometheus.MustNewConstSummaryWithCreatedTimestamp(
				desc,
				m.summaryCount,
				m.summarySum,
				m.summaryQuantiles,
				m.timestamp,
				labels...,
			)
		default:
			logrus.WithField("type", cc.mapping.Type).Warn("unknown metric type")
		}
		return true
	})
}
