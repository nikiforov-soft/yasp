package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
)

var (
	ErrAlreadyListening = errors.New("already listening")
)

type Service interface {
	Observe(key Key, value float64, labels prometheus.Labels) error
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type service struct {
	tlsCertificateFile string
	tlsPrivateKey      string
	endpoint           string
	listening          atomic.Bool
	server             *http.Server
	metricsMapping     []*config.MetricsMapping
	metricByName       map[string]any
	metricByNameLock   sync.Mutex
}

func NewService(config config.Metrics) Service {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}
	mux.Handle(config.Endpoint, promhttp.Handler())

	return &service{
		tlsCertificateFile: config.TLS.CertificateFile,
		tlsPrivateKey:      config.TLS.PrivateKeyFile,
		endpoint:           config.Endpoint,
		server:             server,
		metricByName:       make(map[string]any),
		metricsMapping:     config.MetricsMapping,
	}
}

func (s *service) Observe(key Key, value float64, labels prometheus.Labels) error {
	logrus.
		WithField("key", key.String()).
		WithField("value", value).
		Info("observing metric value changes")

	var mapping *config.MetricsMapping
	for _, m := range s.metricsMapping {
		if strings.EqualFold(m.Name, key.Name) &&
			strings.EqualFold(m.Namespace, key.Namespace) &&
			strings.EqualFold(m.Subsystem, key.Subsystem) {
			mapping = m
			break
		}
	}
	if mapping == nil {
		logrus.WithField("name", key).Warn("metrics mapping not found")
		return nil
	}

	if len(labels) != len(mapping.Labels) {
		return fmt.Errorf("metrics: mismatched label name/value count for %s expected %d got %d", key, len(mapping.Labels), len(labels))
	}

	for _, k := range mapping.Labels {
		if _, exists := labels[k]; !exists {
			return fmt.Errorf("metrics: missing required label %s for: %s", k, key)
		}
	}

	for _, k := range labels {
		if !slices.Contains(mapping.Labels, k) {
			return fmt.Errorf("metrics: provided unknown label %s for: %s", k, key)
		}
	}

	metric, err := s.computeMetricVec(mapping)
	if err != nil {
		return err
	}

	switch m := metric.(type) {
	case *prometheus.CounterVec:
		m.With(labels).Add(value)
	case *prometheus.GaugeVec:
		m.With(labels).Set(value)
	case *prometheus.SummaryVec:
		m.With(labels).Observe(value)
	case *prometheus.HistogramVec:
		m.With(labels).Observe(value)
	default:
		return fmt.Errorf("metrics: unsupported type: %s for: %s", key, mapping.Type)
	}
	return nil
}

func (s *service) ListenAndServe() error {
	if !s.listening.CompareAndSwap(false, true) {
		return ErrAlreadyListening
	}
	if s.tlsCertificateFile != "" && s.tlsPrivateKey != "" {
		logrus.Infof("metrics: listening on: https://%s/%s", s.server.Addr, strings.TrimPrefix(s.endpoint, "/"))
		return s.server.ListenAndServeTLS(s.tlsCertificateFile, s.tlsPrivateKey)
	}
	logrus.Infof("metrics: listening on: http://%s/%s", s.server.Addr, strings.TrimPrefix(s.endpoint, "/"))
	return s.server.ListenAndServe()
}

func (s *service) Shutdown(ctx context.Context) error {
	if !s.listening.Load() {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *service) computeMetricVec(mapping *config.MetricsMapping) (any, error) {
	s.metricByNameLock.Lock()
	defer s.metricByNameLock.Unlock()

	metric, exists := s.metricByName[mapping.Name]
	if !exists {
		switch strings.ToLower(mapping.Type) {
		case "counter":
			metric = promauto.NewCounterVec(prometheus.CounterOpts{
				Namespace: mapping.Namespace,
				Subsystem: mapping.Subsystem,
				Name:      mapping.Name,
				Help:      mapping.Description,
			}, mapping.Labels)
		case "gauge":
			metric = promauto.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: mapping.Namespace,
				Subsystem: mapping.Subsystem,
				Name:      mapping.Name,
				Help:      mapping.Description,
			}, mapping.Labels)
		case "summary":
			metric = promauto.NewSummaryVec(prometheus.SummaryOpts{
				Namespace:  mapping.Namespace,
				Subsystem:  mapping.Subsystem,
				Name:       mapping.Name,
				Help:       mapping.Description,
				Objectives: mapping.Objectives,
				MaxAge:     mapping.MaxAge,
				AgeBuckets: mapping.AgeBuckets,
				BufCap:     mapping.BufCap,
			}, mapping.Labels)
		case "histogram":
			metric = promauto.NewHistogramVec(prometheus.HistogramOpts{
				Namespace:                       mapping.Namespace,
				Subsystem:                       mapping.Subsystem,
				Name:                            mapping.Name,
				Help:                            mapping.Description,
				Buckets:                         mapping.Buckets,
				NativeHistogramBucketFactor:     mapping.NativeHistogramBucketFactor,
				NativeHistogramZeroThreshold:    mapping.NativeHistogramZeroThreshold,
				NativeHistogramMaxBucketNumber:  mapping.NativeHistogramMaxBucketNumber,
				NativeHistogramMinResetDuration: mapping.NativeHistogramMinResetDuration,
				NativeHistogramMaxZeroThreshold: mapping.NativeHistogramMaxZeroThreshold,
			}, mapping.Labels)
		default:
			return nil, fmt.Errorf("prometheus: unsupported type: %s", mapping.Type)
		}
		s.metricByName[mapping.Name] = metric
	}
	return metric, nil
}
