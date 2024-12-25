package config

import (
	"time"
)

type MetricsMapping struct {
	// Counter, Gauge
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Namespace   string   `yaml:"namespace"`
	Subsystem   string   `yaml:"subsystem"`
	Labels      []string `yaml:"labels"`
	// Histogram
	Buckets                         []float64     `yaml:"buckets"`
	NativeHistogramBucketFactor     float64       `yaml:"nativeHistogramBucketFactor"`
	NativeHistogramZeroThreshold    float64       `yaml:"nativeHistogramZeroThreshold"`
	NativeHistogramMaxBucketNumber  uint32        `yaml:"nativeHistogramMaxBucketNumber"`
	NativeHistogramMinResetDuration time.Duration `yaml:"nativeHistogramMinResetDuration"`
	NativeHistogramMaxZeroThreshold float64       `yaml:"nativeHistogramMaxZeroThreshold"`
	// Summary
	Objectives map[float64]float64 `yaml:"objectives"`
	MaxAge     time.Duration       `yaml:"maxAge"`
	AgeBuckets uint32              `yaml:"ageBuckets"`
	BufCap     uint32              `yaml:"bufCap"`
	// Additional context
	Type string `yaml:"type"`
}
