package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/process"
	"github.com/sirupsen/logrus"
	"go.uber.org/automaxprocs/maxprocs"

	// Side effect imports
	_ "github.com/nikiforov-soft/yasp/input/impl"
	_ "github.com/nikiforov-soft/yasp/input/transform/impl"
	_ "github.com/nikiforov-soft/yasp/output/impl"
	_ "github.com/nikiforov-soft/yasp/output/transform/impl"
	_ "github.com/nikiforov-soft/yasp/sensor/impl"
)

var (
	configPath string
)

func main() {
	flag.StringVar(&configPath, "config", "./config.yaml", "Path to the config.yaml")
	flag.Parse()
	started := time.Now()
	logrus.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339,
	})

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

	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		logrus.Info("detected kubernetes waiting for istio proxy")
		if err := waitForIstioProxy(); err != nil {
			logrus.
				WithError(err).
				Error("failed to wait for istio proxy")
			return
		}
	}

	processService, err := process.NewService(context.Background(), conf.Sensors)
	if err != nil {
		logrus.
			WithError(err).
			Error("failed to initialize process service")
		return
	}

	logrus.Infof("initialization completed in %s", time.Since(started).String())

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGTERM, syscall.SIGINT)

	<-shutdownChan
	logrus.Info("Shutting down")

	if err := processService.Close(); err != nil {
		logrus.WithError(err).Error("failed to shutdown process service")
	}
}
