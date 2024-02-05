package prometheus

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/output"
	"github.com/nikiforov-soft/yasp/template"
)

type promeheus struct {
	config       *config.Prometheus
	metricByName map[string]any
}

func newPrometheus(_ context.Context, config *config.Prometheus) (*promeheus, error) {
	return &promeheus{
		config:       config,
		metricByName: make(map[string]any),
	}, nil
}

func (p *promeheus) Publish(_ context.Context, data *output.Data) error {
	logrus.
		WithField("data", string(data.Data)).
		WithField("properties", data.Properties).
		Debug("prometheus data")

	for _, mapping := range p.config.MetricsMapping {
		if mapping.Condition != "" {
			conditionBytes, err := template.Execute("mapping.condition", mapping.Condition, data)
			if err != nil {
				return fmt.Errorf("prometheus: failed to process condition template: %w", err)
			}

			conditionValue, err := strconv.ParseBool(string(conditionBytes))
			if err != nil {
				return fmt.Errorf("prometheus: failed to parse condition as boolean: %w", err)
			}

			if !conditionValue {
				continue
			}
		}

		value, err := template.Execute("mapping.value", mapping.Value, data)
		if err != nil {
			return fmt.Errorf("prometheus: failed to process value template: %w", err)
		}

		namespace, err := template.Execute("mapping.namespace", mapping.Namespace, data)
		if err != nil {
			return fmt.Errorf("prometheus: failed to process namespace template: %w", err)
		}

		subsystem, err := template.Execute("mapping.subsystem", mapping.Subsystem, data)
		if err != nil {
			return fmt.Errorf("prometheus: failed to process subsystem template: %w", err)
		}

		name, err := template.Execute("mapping.name", mapping.Name, data)
		if err != nil {
			return fmt.Errorf("prometheus: failed to process name template: %w", err)
		}

		description, err := template.Execute("mapping.description", mapping.Description, data)
		if err != nil {
			return fmt.Errorf("prometheus: failed to process description template: %w", err)
		}

		mappingType, err := template.Execute("mapping.type", mapping.Type, data)
		if err != nil {
			return fmt.Errorf("prometheus: failed to process type template: %w", err)
		}

		labelKeys := make([]string, 0, len(mapping.Labels))
		labelValues := make([]string, 0, len(mapping.Labels))
		for k, v := range mapping.Labels {
			labelKey, err := template.Execute("mapping.label.key", k, data)
			if err != nil {
				return fmt.Errorf("prometheus: failed to process label key template: %w", err)
			}

			labelValue, err := template.Execute("mapping.label.value", v, data)
			if err != nil {
				return fmt.Errorf("prometheus: failed to process label value template: %w", err)
			}

			labelKeys = append(labelKeys, string(labelKey))
			labelValues = append(labelValues, string(labelValue))
		}

		metric, exists := p.metricByName[string(name)]
		if !exists {
			switch strings.ToLower(string(mappingType)) {
			case "counter":
				metric = promauto.NewCounterVec(prometheus.CounterOpts{
					Namespace: string(namespace),
					Subsystem: string(subsystem),
					Name:      string(name),
					Help:      string(description),
				}, labelKeys)
			case "gauge":
				metric = promauto.NewGaugeVec(prometheus.GaugeOpts{
					Namespace: string(namespace),
					Subsystem: string(subsystem),
					Name:      string(name),
					Help:      string(description),
				}, labelKeys)
			case "summary":
				metric = promauto.NewSummaryVec(prometheus.SummaryOpts{
					Namespace:  string(namespace),
					Subsystem:  string(subsystem),
					Name:       string(name),
					Help:       string(description),
					Objectives: mapping.Objectives,
					MaxAge:     mapping.MaxAge,
					AgeBuckets: mapping.AgeBuckets,
					BufCap:     mapping.BufCap,
				}, labelKeys)
			case "histogram":
				metric = promauto.NewHistogramVec(prometheus.HistogramOpts{
					Namespace:                       string(namespace),
					Subsystem:                       string(subsystem),
					Name:                            string(name),
					Help:                            string(description),
					Buckets:                         mapping.Buckets,
					NativeHistogramBucketFactor:     mapping.NativeHistogramBucketFactor,
					NativeHistogramZeroThreshold:    mapping.NativeHistogramZeroThreshold,
					NativeHistogramMaxBucketNumber:  mapping.NativeHistogramMaxBucketNumber,
					NativeHistogramMinResetDuration: mapping.NativeHistogramMinResetDuration,
					NativeHistogramMaxZeroThreshold: mapping.NativeHistogramMaxZeroThreshold,
				}, labelKeys)
			default:
				return fmt.Errorf("prometheus: unsupported type: %s", string(mappingType))
			}
			p.metricByName[string(name)] = metric
		}

		floatValue, err := template.AsNumber(value)
		if err != nil {
			return fmt.Errorf("prometheus: failed to parse value as float64: %s - %w", string(value), err)
		}

		switch m := metric.(type) {
		case *prometheus.CounterVec:
			m.WithLabelValues(labelValues...).Add(floatValue)
		case *prometheus.GaugeVec:
			m.WithLabelValues(labelValues...).Set(floatValue)
		case *prometheus.SummaryVec:
			m.WithLabelValues(labelValues...).Observe(floatValue)
		case *prometheus.HistogramVec:
			m.WithLabelValues(labelValues...).Observe(floatValue)
		default:
			return fmt.Errorf("prometheus: unsupported type: %s", string(mappingType))
		}
	}
	return nil
}

func (p *promeheus) Close(_ context.Context) error {
	return nil
}

func init() {
	err := output.RegisterOutput("prometheus", func(ctx context.Context, config *config.Output) (output.Output, error) {
		return newPrometheus(ctx, config.Prometheus)
	})
	if err != nil {
		panic(err)
	}
}
