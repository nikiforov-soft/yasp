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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/internal/syncx"
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
	closeCtx              context.Context
	closeFn               context.CancelFunc
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
	closeCtx, closeFn := context.WithCancel(context.Background())
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}
	mux.Handle(config.Endpoint, promhttp.Handler())

	s := &service{
		closeCtx:           closeCtx,
		closeFn:            closeFn,
		tlsCertificateFile: config.TLS.CertificateFile,
		tlsPrivateKey:      config.TLS.PrivateKeyFile,
		endpoint:           config.Endpoint,
		stalenessInterval:  config.StalenessInterval,
		server:             server,
		metricsMapping:     config.MetricsMapping,
		collectorByMetric:  make(map[string]*collector),
	}

	if config.StalenessInterval != 0 {
		go s.startTicker()
	}

	return s
}

func (s *service) startTicker() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-s.closeCtx.Done():
			return
		case <-ticker.C:
			s.prune()
		}
	}
}

func (s *service) prune() {
	s.collectorByMetricLock.Lock()
	defer s.collectorByMetricLock.Unlock()

	for k, c := range s.collectorByMetric {
		var metricsToRemove []string
		c.metrics.Range(func(key string, value *metric) bool {
			if time.Since(value.timestamp) > s.stalenessInterval {
				metricsToRemove = append(metricsToRemove, key)
			}
			return true
		})

		for _, key := range metricsToRemove {
			c.metrics.Delete(key)

			logrus.
				WithField("key", key).
				Debug("stale metric removed")
		}

		var metricsLen = 0
		c.metrics.Range(func(key string, value *metric) bool {
			metricsLen++
			return true
		})

		if metricsLen == 0 {
			delete(s.collectorByMetric, k)
			prometheus.Unregister(c)
		}
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

	if len(labels) != len(mapping.Labels) {
		return fmt.Errorf("metrics: mismatched label name/value count for %s expected %d got %d", key, len(mapping.Labels), len(labels))
	}

	for _, k := range mapping.Labels {
		if _, exists := labels[k]; !exists {
			return fmt.Errorf("metrics: missing required label %s for: %s", k, key)
		}
	}

	for k := range labels {
		if !slices.Contains(mapping.Labels, k) {
			return fmt.Errorf("metrics: provided unknown label %s for: %s", k, key)
		}
	}

	c, err := s.computeCollector(mapping)
	if err != nil {
		return err
	}

	c.metrics.Store(key.String(), newMetric(mapping, value, labels))
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
	s.closeFn()
	return s.server.Shutdown(ctx)
}

func (s *service) computeCollector(mapping *config.MetricsMapping) (*collector, error) {
	s.collectorByMetricLock.Lock()
	defer s.collectorByMetricLock.Unlock()

	c, exists := s.collectorByMetric[mapping.Name]
	if !exists {
		c = &collector{
			metricsMapping: mapping,
			metrics:        &syncx.Map[string, *metric]{},
		}
		if err := prometheus.Register(c); err != nil {
			return nil, fmt.Errorf("failed to register metrics collector: %w", err)
		}
		s.collectorByMetric[mapping.Name] = c
	}
	return c, nil
}
