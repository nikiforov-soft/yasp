package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Metrics Metrics   `yaml:"metrics"`
	Sensors []*Sensor `yaml:"sensors"`
}

type configWithOptionalMetrics struct {
	Metrics *Metrics  `yaml:"metrics,omitempty"`
	Sensors []*Sensor `yaml:"sensors,omitempty"`
}

func Load(filePath string) (*Config, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		return loadFile[Config](filePath)
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path of %s: %w", filePath, err)
	}

	logrus.WithField("path", absFilePath).Info("loading config from directory recursively...")

	var config Config
	var metricsLoaded bool
	err = filepath.WalkDir(absFilePath, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(filePath, ".yaml") && strings.HasSuffix(filePath, ".yml") {
			return nil
		}

		relativeFilePath, err := filepath.Rel(absFilePath, filePath)
		if err != nil {
			return fmt.Errorf("failed to resolve relative path of %s: %w", filePath, err)
		}

		logrus.WithField("file", relativeFilePath).Info("loading config")

		c, err := loadFile[configWithOptionalMetrics](filePath)
		if err != nil {
			return fmt.Errorf("failed to load config file %s: %w", filePath, err)
		}
		if c.Metrics != nil {
			if metricsLoaded {
				logrus.WithField("file", filePath).Warn("loaded multiple metrics configs, overriding")
			}
			metricsLoaded = true
			config.Metrics = *c.Metrics
		}
		config.Sensors = append(config.Sensors, c.Sensors...)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func loadFile[T any](filePath string) (*T, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config *T
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}
	return config, nil
}
