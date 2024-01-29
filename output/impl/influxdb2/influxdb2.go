package influxdb2

import (
	"context"
	"errors"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/output"
	"github.com/nikiforov-soft/yasp/template"
)

var (
	eventsProcessedCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name:      "output_events_published",
		Help:      "The amount of events influxdb output published.",
		Namespace: "yasp",
		Subsystem: "influxdb2",
	}, []string{"bucket"})
)

type influxdb struct {
	config *config.InfluxDb2
	client influxdb2.Client
}

func newInfluxDb2(ctx context.Context, config *config.InfluxDb2) (*influxdb, error) {
	options := influxdb2.DefaultOptions()
	if config.UseGZip {
		options.UseGZip()
	}
	if config.BatchSize != 0 {
		options.SetBatchSize(config.BatchSize)
	}
	client := influxdb2.NewClientWithOptions(config.Url, config.AuthToken, options)
	defer client.Close()

	ping, err := client.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("influxdb2 output: failed to ping server: %w", err)
	}

	if !ping {
		return nil, errors.New("influxdb2 output: failed to ping server")
	}

	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("influxdb2 output: failed to health check: %w", err)
	}

	if health.Status != domain.HealthCheckStatusPass {
		return nil, fmt.Errorf("influxdb2 output: influxdb is not ready: %s", health.Status)
	}

	ready, err := client.Ready(ctx)
	if err != nil {
		return nil, fmt.Errorf("influxdb2 output: failed to ready check: %w", err)
	}

	if ready.Status == nil || *ready.Status != domain.ReadyStatusReady {
		return nil, fmt.Errorf("influxdb2 output: server is not ready: %+v", ready.Status)
	}
	logrus.Info("influxdb2 output: connected to the server")

	return &influxdb{
		config: config,
		client: client,
	}, nil
}

func (i *influxdb) Publish(ctx context.Context, data *output.Data) error {
	writeApi := i.client.WriteAPIBlocking(i.config.OrganizationId, i.config.Bucket)

	measurement, err := template.Execute("measurement", i.config.Measurement, data)
	if err != nil {
		return err
	}

	point := influxdb2.NewPointWithMeasurement(string(measurement))
	point.SetTime(time.Now())

	for tagKey, tagValue := range i.config.TagMapping {
		if tagValue == "-" || tagValue == "_" {
			continue
		}

		value, err := template.Execute(tagKey, tagValue, data)
		if err != nil {
			return err
		}

		point.AddTag(tagKey, string(value))
	}

	for fieldKey, fieldValue := range i.config.FieldMapping {
		if fieldValue == "-" || fieldValue == "_" {
			continue
		}

		value, err := template.Execute(fieldKey, fieldValue, data)
		if err != nil {
			return err
		}

		if template.IsNumber(value) {
			float64Value, err := template.AsNumber(value)
			if err != nil {
				return fmt.Errorf("influxdb2 output: failed to parse %s as number: %w", string(value), err)
			}
			point.AddField(fieldKey, float64Value)
			continue
		}

		point.AddField(fieldKey, value)
	}

	if err := writeApi.WritePoint(ctx, point); err != nil {
		return fmt.Errorf("influxdb2 output: failed to write point: %w", err)
	}

	eventsProcessedCounter.WithLabelValues(i.config.Bucket).Inc()

	return nil
}

func (i *influxdb) Close(_ context.Context) error {
	writeApi := i.client.WriteAPI(i.config.OrganizationId, i.config.Bucket)
	writeApi.Flush()
	i.client.Close()
	return nil
}

func init() {
	err := output.RegisterOutput("influxdb2", func(ctx context.Context, config *config.Output) (output.Output, error) {
		return newInfluxDb2(ctx, config.InfluxDb2)
	})
	if err != nil {
		panic(err)
	}
}
