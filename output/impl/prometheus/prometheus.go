package prometheus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/output"
	"github.com/nikiforov-soft/yasp/template"
)

type promeheus struct {
	config       *config.Prometheus
	server       *http.Server
	metricByName map[string]prometheus.Metric
}

func newPrometheus(_ context.Context, config *config.Prometheus) (*promeheus, error) {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}
	mux.Handle(config.Endpoint, promhttp.Handler())

	go func() {
		var err error
		if config.TLS.CertificateFile != "" && config.TLS.PrivateKeyFile != "" {
			logrus.Infof("prometheus output: listening on: https://%s/%s", config.ListenAddr, strings.TrimPrefix(config.Endpoint, "/"))
			err = server.ListenAndServeTLS(config.TLS.CertificateFile, config.TLS.PrivateKeyFile)
		} else {
			logrus.Infof("prometheus output: listening on: http://%s/%s", config.ListenAddr, strings.TrimPrefix(config.Endpoint, "/"))
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Error("prometheus: failed to listen and serve")
		}
	}()

	return &promeheus{
		config:       config,
		server:       server,
		metricByName: make(map[string]prometheus.Metric),
	}, nil
}

func (p *promeheus) Publish(_ context.Context, data *output.Data) error {
	logrus.
		WithField("data", string(data.Data)).
		WithField("properties", data.Properties).
		Debug("prometheus data")

	for _, mapping := range p.config.MetricsMapping {
		conditionBytes, err := template.Execute(mapping.Name, mapping.Condition, data)
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

		value, err := template.Execute(mapping.Name, mapping.Value, data)
		if err != nil {
			return fmt.Errorf("prometheus: failed to process value template: %w", err)
		}

		metric, exists := p.metricByName[mapping.Name]
		if !exists {
			switch strings.ToLower(mapping.Type) {
			case "counter":
				metric = promauto.NewCounter(prometheus.CounterOpts{
					Namespace:   mapping.Namespace,
					Subsystem:   mapping.Subsystem,
					Name:        mapping.Name,
					Help:        mapping.Description,
					ConstLabels: mapping.Labels,
				})
			case "gauge":
				metric = promauto.NewGauge(prometheus.GaugeOpts{
					Namespace:   mapping.Namespace,
					Subsystem:   mapping.Subsystem,
					Name:        mapping.Name,
					Help:        mapping.Description,
					ConstLabels: mapping.Labels,
				})
			case "summary":
				metric = promauto.NewSummary(prometheus.SummaryOpts{
					Namespace:   mapping.Namespace,
					Subsystem:   mapping.Subsystem,
					Name:        mapping.Name,
					Help:        mapping.Description,
					ConstLabels: mapping.Labels,
				})
			case "histogram":
				metric = promauto.NewHistogram(prometheus.HistogramOpts{
					Namespace:   mapping.Namespace,
					Subsystem:   mapping.Subsystem,
					Name:        mapping.Name,
					Help:        mapping.Description,
					ConstLabels: mapping.Labels,
				})
			default:
				return fmt.Errorf("prometheus: unsupported type: %s", mapping.Type)
			}
			p.metricByName[mapping.Name] = metric
		}

		switch m := metric.(type) {
		case prometheus.Gauge:
			v, err := template.AsNumber(value)
			if err != nil {
				return fmt.Errorf("prometheus: failed to parse gauge value as float64: %s - %w", string(value), err)
			}
			m.Set(v)
		case prometheus.Summary:
			v, err := template.AsNumber(value)
			if err != nil {
				return fmt.Errorf("prometheus: failed to parse summary value as float64: %s - %w", string(value), err)
			}
			m.Observe(v)
		case prometheus.Histogram:
			v, err := template.AsNumber(value)
			if err != nil {
				return fmt.Errorf("prometheus: failed to parse histogram value as float64: %s - %w", string(value), err)
			}
			m.Observe(v)
		case prometheus.Counter:
			v, err := template.AsNumber(value)
			if err != nil {
				return fmt.Errorf("prometheus: failed to parse counter value as float64: %s - %w", string(value), err)
			}
			m.Add(v)
		default:
			return fmt.Errorf("prometheus: unsupported type: %s", mapping.Type)
		}
	}
	return nil
}

func (p *promeheus) Close(ctx context.Context) error {
	return p.server.Shutdown(ctx)
}

func init() {
	err := output.RegisterOutput("prometheus", func(ctx context.Context, config *config.Output) (output.Output, error) {
		return newPrometheus(ctx, config.Prometheus)
	})
	if err != nil {
		panic(err)
	}
}
