package influxdb2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
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
		"ToNumber": func(value any) interface{} {
			var stringValue string
			switch value := value.(type) {
			case []byte:
				stringValue = string(value)
			case string:
				stringValue = value
			default:
				return fmt.Sprintf("%s", value)
			}

			if float64Value, err := strconv.ParseFloat(stringValue, 64); err == nil {
				return float64Value
			}

			if int64Value, err := strconv.ParseInt(stringValue, 10, 64); err == nil {
				return int64Value
			}

			return stringValue
		},
	}
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
		return nil, fmt.Errorf("influxdb2: failed to ping server: %w", err)
	}

	if !ping {
		return nil, errors.New("influxdb2: failed to ping server")
	}

	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("influxdb2: failed to health check: %w", err)
	}

	if health.Status != domain.HealthCheckStatusPass {
		return nil, fmt.Errorf("influxdb2: influxdb is not ready: %s", health.Status)
	}

	ready, err := client.Ready(ctx)
	if err != nil {
		return nil, fmt.Errorf("influxdb2: failed to ready check: %w", err)
	}

	if ready.Status == nil || *ready.Status != domain.ReadyStatusReady {
		return nil, fmt.Errorf("influxdb2: server is not ready: %+v", ready.Status)
	}

	return &influxdb{
		config: config,
		client: client,
	}, nil
}

func (i *influxdb) Publish(_ context.Context, data *output.Data) error {
	writeApi := i.client.WriteAPI(i.config.OrganizationId, i.config.Bucket)
	measurement, err := templateProcess("measurement", i.config.Measurement, data)
	if err != nil {
		return err
	}

	point := influxdb2.NewPointWithMeasurement(string(measurement))
	point.SetTime(time.Now())

	for tagKey, tagValue := range i.config.TagMapping {
		if tagValue == "-" || tagValue == "_" {
			continue
		}

		value, err := templateProcess(tagKey, tagValue, data)
		if err != nil {
			return err
		}

		point.AddTag(tagKey, string(value))
	}

	for fieldKey, fieldValue := range i.config.FieldMapping {
		if fieldValue == "-" || fieldValue == "_" {
			continue
		}

		value, err := templateProcess(fieldKey, fieldValue, data)
		if err != nil {
			return err
		}

		if isNumber(value) {
			float64Value, err := asNumber(value)
			if err != nil {
				return fmt.Errorf("failed to parse %s as number: %w", string(value), err)
			}
			point.AddField(fieldKey, float64Value)
			continue
		}

		point.AddField(fieldKey, value)
	}

	writeApi.WritePoint(point)
	writeApi.Flush()
	return nil
}

func templateProcess(templateKey string, templateValue string, data any) ([]byte, error) {
	tmpl, err := template.New(templateKey).Funcs(funcsMap).Parse(templateValue)
	if err != nil {
		return nil, fmt.Errorf("influxdb2: failed to parse glob: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("influxdb2: failed to execute %s mapping template: - %w", templateKey, err)
	}
	return buf.Bytes(), nil
}

func isNumber(n []byte) bool {
	if len(n) > 0 && n[0] == '-' {
		n = n[1:]
	}
	if len(n) == 0 {
		return false
	}
	var point bool
	for _, c := range n {
		if '0' <= c && c <= '9' {
			continue
		}
		if c == '.' && len(n) > 1 && !point {
			point = true
			continue
		}
		return false
	}
	return true
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

func (i *influxdb) Close(_ context.Context) error {
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
