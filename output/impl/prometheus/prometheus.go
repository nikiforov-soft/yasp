package influxdb2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/output"
)

var (
	funcsMap = map[string]any{
		"ToLower":    strings.ToLower,
		"ToUpper":    strings.ToUpper,
		"TrimSpaces": strings.TrimSpace,
		"TrimPrefix": strings.TrimPrefix,
		"TrimSuffix": strings.TrimSuffix,
		"ToNumber": func(value any) any {
			var stringValue string
			switch value := value.(type) {
			case uint8:
				return value
			case uint16:
				return value
			case uint32:
				return value
			case uint64:
				return value
			case int8:
				return value
			case int16:
				return value
			case int32:
				return value
			case int64:
				return value
			case float64:
				return value
			case float32:
				return value
			case []byte:
				stringValue = string(value)
			case string:
				stringValue = value
			default:
				return fmt.Sprintf("%v", value)
			}

			if float64Value, err := strconv.ParseFloat(stringValue, 64); err == nil {
				return float64Value
			}

			if int64Value, err := strconv.ParseInt(stringValue, 10, 64); err == nil {
				return int64Value
			}

			return stringValue
		},
		"Split": strings.Split,
		"Last": func(values []string) string {
			if len(values) == 0 {
				return ""
			}
			return values[len(values)-1]
		},
		"Quote": strconv.Quote,
	}
)

type promeheus struct {
	config       *config.Prometheus
	server       *http.Server
	metricByName map[string]prometheus.Metric
}

func newPrometheus(_ context.Context, config *config.Prometheus) (*promeheus, error) {
	logrus.Info("prometheus output: listening")

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}
	mux.Handle(config.Endpoint, promhttp.Handler())

	go func() {
		var err error
		if config.TLS.CertificateFile != "" && config.TLS.PrivateKeyFile != "" {
			err = server.ListenAndServeTLS(config.TLS.CertificateFile, config.TLS.PrivateKeyFile)
		} else {
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
		Info("prometheus data")

	for _, mapping := range p.config.MetricsMapping {
		conditionBytes, err := templateProcess(mapping.Name, mapping.Condition, data)
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

		value, err := templateProcess(mapping.Name, mapping.Value, data)
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
			v, err := asNumber(value)
			if err != nil {
				return fmt.Errorf("prometheus: failed to parse gauge value as float64: %s - %w", string(value), err)
			}
			m.Set(v)
		case prometheus.Summary:
			v, err := asNumber(value)
			if err != nil {
				return fmt.Errorf("prometheus: failed to parse summary value as float64: %s - %w", string(value), err)
			}
			m.Observe(v)
		case prometheus.Histogram:
			v, err := asNumber(value)
			if err != nil {
				return fmt.Errorf("prometheus: failed to parse histogram value as float64: %s - %w", string(value), err)
			}
			m.Observe(v)
		case prometheus.Counter:
			v, err := asNumber(value)
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

func templateProcess(templateKey, templateValue string, data any) ([]byte, error) {
	tmpl, err := template.New(templateKey).Funcs(funcsMap).Parse(templateValue)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute %s template: - %w", templateKey, err)
	}
	return buf.Bytes(), nil
}

func asNumber(data []byte) (float64, error) {
	if float64Value, err := strconv.ParseFloat(string(data), 64); err == nil {
		return float64Value, nil
	}
	if int64Value, err := strconv.ParseInt(string(data), 10, 64); err == nil {
		return float64(int64Value), nil
	}
	return 0, fmt.Errorf("failed to parse %s as number", string(data))
}

func init() {
	err := output.RegisterOutput("prometheus", func(ctx context.Context, config *config.Output) (output.Output, error) {
		return newPrometheus(ctx, config.Prometheus)
	})
	if err != nil {
		panic(err)
	}
}
