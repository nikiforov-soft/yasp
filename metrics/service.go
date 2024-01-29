package metrics

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
)

var (
	ErrAlreadyListening = errors.New("already listening")
)

type Service interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

type service struct {
	tlsCertificateFile string
	tlsPrivateKey      string
	endpoint           string
	listening          atomic.Bool
	server             *http.Server
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
	}
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
