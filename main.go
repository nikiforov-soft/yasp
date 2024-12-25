package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/metrics"
	"github.com/nikiforov-soft/yasp/process"

	// Side effect imports
	_ "github.com/nikiforov-soft/yasp/device/impl"
	_ "github.com/nikiforov-soft/yasp/input/impl"
	_ "github.com/nikiforov-soft/yasp/input/transform/impl"
	_ "github.com/nikiforov-soft/yasp/output/impl"
	_ "github.com/nikiforov-soft/yasp/output/transform/impl"
)

// Injected during build time
var (
	version string = "dev"
	commit  string = "HEAD"
	date    string = time.Now().Format(time.RFC3339)
)

var (
	configPath string
	verbose    bool
)

func main() {
	flag.StringVar(&configPath, "config", "./config.yaml", "Path to the config.yaml")
	flag.BoolVar(&verbose, "verbose", false, "Enables debug logging")
	flag.Parse()

	logrus.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339,
	})
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.
		WithField("version", version).
		WithField("commit", commit).
		WithField("date", date).
		Info("initializing...")

	if _, err := maxprocs.Set(maxprocs.Logger(logrus.Printf)); err != nil {
		logrus.
			WithError(err).
			Error("failed to set maxprocs")
		return
	}

	logrus.Info("initializing configuration...")
	conf, err := config.Load(configPath)
	if err != nil {
		logrus.
			WithError(err).
			Error("failed to load config")
		return
	}

	metricsService := metrics.NewService(conf.Metrics)
	if conf.Metrics.Enabled {
		go func() {
			if err := metricsService.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logrus.WithError(err).Error("metrics failed to listen and serve")
			}
		}()
	}

	processService, err := process.NewService(context.Background(), metricsService, conf.Sensors)
	if err != nil {
		logrus.
			WithError(err).
			Error("failed to initialize process service")
		return
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGTERM, syscall.SIGINT)

	<-shutdownChan
	logrus.Info("Shutting down...")

	logrus.Info("Shutting down process service...")
	if err := processService.Close(); err != nil {
		logrus.WithError(err).Error("failed to shutdown process service")
	}

	logrus.Info("Shutting down metrics service...")
	if err := metricsService.Shutdown(context.Background()); err != nil {
		logrus.WithError(err).Error("failed to shutdown metrics service")
	}
}
