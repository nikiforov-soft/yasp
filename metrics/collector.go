package metrics

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/internal/syncx"
)

type collector struct {
	closeCtx          context.Context
	closeCtxCancel    context.CancelFunc
	desc              *prometheus.Desc
	metricVec         any
	metrics           *syncx.Map[string, metricHistory]
	metricsMapping    *config.MetricsMapping
	stalenessInterval time.Duration
}

func newCollector(metricsMapping *config.MetricsMapping, stalenessInterval time.Duration) (*collector, error) {
	metricVec, err := newMetricVec(metricsMapping)
	if err != nil {
		return nil, err
	}

	closeCtx, closeCtxCancel := context.WithCancel(context.Background())
	c := &collector{
		closeCtx:          closeCtx,
		closeCtxCancel:    closeCtxCancel,
		desc:              newDesc(metricsMapping),
		metricVec:         metricVec,
		metrics:           &syncx.Map[string, metricHistory]{},
		metricsMapping:    metricsMapping,
		stalenessInterval: stalenessInterval,
	}

	if err := prometheus.Register(c); err != nil {
		return nil, fmt.Errorf("failed to register metrics collector: %w", err)
	}

	if stalenessInterval > 0 {
		go c.runMetricsCleanupTask()
	}

	return c, nil
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.metrics.Range(func(_ string, m metricHistory) bool {
		metricValue, err := c.getMetric(m.labels)
		if err != nil {
			ch <- prometheus.NewInvalidMetric(c.desc, err)
		} else {
			ch <- prometheus.NewMetricWithTimestamp(m.lastUpdatedAt, metricValue)
		}
		return true
	})
}

func (c *collector) Close() {
	c.closeCtxCancel()
	prometheus.Unregister(c)
}

func (c *collector) Observe(value float64, labels prometheus.Labels) error {
	switch m := c.metricVec.(type) {
	case *prometheus.CounterVec:
		m.With(labels).Add(value)
	case *prometheus.GaugeVec:
		m.With(labels).Set(value)
	case *prometheus.HistogramVec:
		m.With(labels).Observe(value)
	case *prometheus.SummaryVec:
		m.With(labels).Observe(value)
	default:
		return fmt.Errorf("unknown metricVec type: %T", c.metricVec)
	}

	c.metrics.Store(computeHash(c.metricsMapping, labels), metricHistory{
		labels:        labels,
		lastUpdatedAt: time.Now(),
	})

	return nil
}

func (c *collector) getMetric(labels prometheus.Labels) (prometheus.Metric, error) {
	switch m := c.metricVec.(type) {
	case *prometheus.CounterVec:
		return m.GetMetricWith(labels)
	case *prometheus.GaugeVec:
		return m.GetMetricWith(labels)
	case *prometheus.HistogramVec:
		return m.MetricVec.GetMetricWith(labels)
	case *prometheus.SummaryVec:
		return m.MetricVec.GetMetricWith(labels)
	default:
		return nil, fmt.Errorf("unsupported metricVec type: %T", m)
	}
}

func (c *collector) runMetricsCleanupTask() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-c.closeCtx.Done():
			return
		case <-ticker.C:
			c.pruneStaleMetrics()
		}
	}
}

func (c *collector) pruneStaleMetrics() {
	var keysToDelete []string
	c.metrics.Range(func(key string, value metricHistory) bool {
		if time.Since(value.lastUpdatedAt) > c.stalenessInterval {
			keysToDelete = append(keysToDelete, key)
		}
		return true
	})

	for _, key := range keysToDelete {
		mh, ok := c.metrics.LoadAndDelete(key)
		if !ok {
			continue
		}

		logrus.
			WithField("name", c.metricsMapping.Name).
			WithField("labels", mh.labels).
			WithField("updatedAt", mh.lastUpdatedAt).
			Debug("stale metric removed")

		switch m := c.metricVec.(type) {
		case *prometheus.CounterVec:
			m.Delete(mh.labels)
		case *prometheus.GaugeVec:
			m.Delete(mh.labels)
		case *prometheus.HistogramVec:
			m.Delete(mh.labels)
		case *prometheus.SummaryVec:
			m.Delete(mh.labels)
		}
	}
}

func newMetricVec(mapping *config.MetricsMapping) (any, error) {
	switch strings.ToLower(mapping.Type) {
	case "counter":
		return prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace:   mapping.Namespace,
			Subsystem:   mapping.Subsystem,
			Name:        mapping.Name,
			ConstLabels: mapping.ConstLabels,
			Help:        mapping.Description,
		}, mapping.Labels), nil
	case "gauge":
		return prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   mapping.Namespace,
			Subsystem:   mapping.Subsystem,
			Name:        mapping.Name,
			ConstLabels: mapping.ConstLabels,
			Help:        mapping.Description,
		}, mapping.Labels), nil
	case "summary":
		return prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:   mapping.Namespace,
			Subsystem:   mapping.Subsystem,
			Name:        mapping.Name,
			Help:        mapping.Description,
			ConstLabels: mapping.ConstLabels,
			Objectives:  mapping.Objectives,
			MaxAge:      mapping.MaxAge,
			AgeBuckets:  mapping.AgeBuckets,
			BufCap:      mapping.BufCap,
		}, mapping.Labels), nil
	case "histogram":
		return prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                       mapping.Namespace,
			Subsystem:                       mapping.Subsystem,
			Name:                            mapping.Name,
			Help:                            mapping.Description,
			ConstLabels:                     mapping.ConstLabels,
			Buckets:                         mapping.Buckets,
			NativeHistogramBucketFactor:     mapping.NativeHistogramBucketFactor,
			NativeHistogramZeroThreshold:    mapping.NativeHistogramZeroThreshold,
			NativeHistogramMaxBucketNumber:  mapping.NativeHistogramMaxBucketNumber,
			NativeHistogramMinResetDuration: mapping.NativeHistogramMinResetDuration,
			NativeHistogramMaxZeroThreshold: mapping.NativeHistogramMaxZeroThreshold,
			NativeHistogramMaxExemplars:     mapping.NativeHistogramMaxExemplars,
			NativeHistogramExemplarTTL:      mapping.NativeHistogramExemplarTTL,
		}, mapping.Labels), nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", mapping.Type)
	}
}

func newDesc(mapping *config.MetricsMapping) *prometheus.Desc {
	return prometheus.NewDesc(
		prometheus.BuildFQName(mapping.Namespace, mapping.Subsystem, mapping.Name),
		mapping.Description,
		mapping.Labels,
		mapping.ConstLabels,
	)
}
