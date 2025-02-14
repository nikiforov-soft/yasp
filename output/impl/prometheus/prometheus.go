package prometheus

import (
	"context"
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/metrics"
	"github.com/nikiforov-soft/yasp/output"
	"github.com/nikiforov-soft/yasp/template"
)

type promeheus struct {
	config         *config.Prometheus
	metricsService metrics.Service
}

func newPrometheus(_ context.Context, config *config.Prometheus, metricsService metrics.Service) (*promeheus, error) {
	return &promeheus{
		config:         config,
		metricsService: metricsService,
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

		labels := make(prometheus.Labels, len(mapping.Labels))
		for k, v := range mapping.Labels {
			labelKey, err := template.Execute("mapping.label.key", k, data)
			if err != nil {
				return fmt.Errorf("prometheus: failed to process label key template: %w", err)
			}

			labelValue, err := template.Execute("mapping.label.value", v, data)
			if err != nil {
				return fmt.Errorf("prometheus: failed to process label value template: %w", err)
			}

			labels[string(labelKey)] = string(labelValue)
		}

		floatValue, err := template.AsNumber(value)
		if err != nil {
			return fmt.Errorf("prometheus: failed to parse value as float64: %s - %w", string(value), err)
		}

		key := metrics.Key{
			Name:      string(name),
			Namespace: string(namespace),
			Subsystem: string(subsystem),
		}

		if err := p.metricsService.Observe(key, floatValue, labels); err != nil {
			return fmt.Errorf("prometheus: failed to process metric: %s - %w", key, err)
		}

	}
	return nil
}

func (p *promeheus) Close(_ context.Context) error {
	return nil
}

func init() {
	err := output.RegisterOutput("prometheus", func(ctx context.Context, config *config.Output, metricsService metrics.Service) (output.Output, error) {
		return newPrometheus(ctx, config.Prometheus, metricsService)
	})
	if err != nil {
		panic(err)
	}
}
