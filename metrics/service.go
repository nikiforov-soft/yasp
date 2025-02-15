package metrics

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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
	tlsCertificateFile    string
	tlsPrivateKey         string
	endpoint              string
	stalenessInterval     time.Duration
	listening             atomic.Bool
	server                *http.Server
	metricsMapping        []*config.MetricsMapping
	collectorByMetric     map[string]*collector
	collectorByMetricLock sync.Mutex
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
		stalenessInterval:  config.StalenessInterval,
		server:             server,
		metricsMapping:     config.MetricsMapping,
		collectorByMetric:  make(map[string]*collector),
	}
}

func (s *service) Observe(key Key, value float64, labels prometheus.Labels) error {
	logrus.
		WithField("key", key.String()).
		WithField("value", value).
		Debug("observing metric value changes")

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

	if err := validateLabels(key, mapping, labels); err != nil {
		return fmt.Errorf("metrics: failed to validate labels: %w", err)
	}

	c, err := s.computeCollector(mapping)
	if err != nil {
		return fmt.Errorf("metrics: failed to get collector: %w", err)
	}

	return c.Observe(value, labels)
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

	s.collectorByMetricLock.Lock()
	defer s.collectorByMetricLock.Unlock()
	for _, c := range s.collectorByMetric {
		c.Close()
	}
	clear(s.collectorByMetric)

	return s.server.Shutdown(ctx)
}

func (s *service) computeCollector(mapping *config.MetricsMapping) (*collector, error) {
	s.collectorByMetricLock.Lock()
	defer s.collectorByMetricLock.Unlock()

	c, exists := s.collectorByMetric[mapping.Name]
	if !exists {
		var err error
		c, err = newCollector(mapping, s.stalenessInterval)
		if err != nil {
			return nil, fmt.Errorf("failed to create collector: %w", err)
		}
		s.collectorByMetric[mapping.Name] = c
	}
	return c, nil
}
