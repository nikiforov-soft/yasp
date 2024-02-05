package process

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/device"
	"github.com/nikiforov-soft/yasp/input"
	inputtransform "github.com/nikiforov-soft/yasp/input/transform"
	"github.com/nikiforov-soft/yasp/output"
	outputtransform "github.com/nikiforov-soft/yasp/output/transform"
)

type Service interface {
	Close() error
}

type service struct {
	sensorGroups []*sensorGroup
	wg           sync.WaitGroup
	cancelFunc   context.CancelFunc
}

func NewService(ctx context.Context, sensors []*config.Sensor) (Service, error) {
	cancellableCtx, cancelFunc := context.WithCancel(ctx)
	s := &service{
		cancelFunc: cancelFunc,
	}
	if err := s.init(cancellableCtx, sensors); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *service) init(ctx context.Context, sensorsConfig []*config.Sensor) error {
	sensorGroups := make([]*sensorGroup, 0, len(sensorsConfig))
	for _, sensorConfig := range sensorsConfig {
		logrus.WithField("sensor", sensorConfig.Name).Info("initializing sensor")

		var inputImpl input.Input
		var inputTransforms []inputtransform.Transform
		var outputs []outputGroup
		var devices []device.Device
		var err error
		if mqtt := sensorConfig.Input.Mqtt; mqtt != nil && mqtt.Enabled {
			inputImpl, err = input.NewInput(ctx, "mqtt", sensorConfig.Input)
			if err != nil {
				return fmt.Errorf("process: failed to initialize input: %w", err)
			}
		} else if memphis := sensorConfig.Input.Memphis; memphis != nil && memphis.Enabled {
			inputImpl, err = input.NewInput(ctx, "memphis", sensorConfig.Input)
			if err != nil {
				return fmt.Errorf("process: failed to initialize input: %w", err)
			}
		}

		for _, transform := range sensorConfig.Input.Transforms {
			inputTransform, err := inputtransform.NewTransform(ctx, transform)
			if err != nil {
				return fmt.Errorf("process: failed to initialize input transform: %s - %w", transform.Name, err)
			}
			inputTransforms = append(inputTransforms, inputTransform)
		}

		for _, o := range sensorConfig.Outputs {
			var outputContainer outputGroup
			if mqtt := o.Mqtt; mqtt != nil && mqtt.Enabled {
				outputImpl, err := output.NewOutput(ctx, "mqtt", o)
				if err != nil {
					return fmt.Errorf("process: failed to initialize mqtt output: %w", err)
				}
				outputContainer.Output = outputImpl
			} else if influxdb2 := o.InfluxDb2; influxdb2 != nil && influxdb2.Enabled {
				outputImpl, err := output.NewOutput(ctx, "influxdb2", o)
				if err != nil {
					return fmt.Errorf("process: failed to initialize influxdb2 output: %w", err)
				}
				outputContainer.Output = outputImpl
			} else if prometheus := o.Prometheus; prometheus != nil && prometheus.Enabled {
				outputImpl, err := output.NewOutput(ctx, "prometheus", o)
				if err != nil {
					return fmt.Errorf("process: failed to initialize prometheus output: %w", err)
				}
				outputContainer.Output = outputImpl
			}

			if outputContainer.Output == nil {
				continue
			}

			for _, transform := range o.Transforms {
				outputTransform, err := outputtransform.NewTransform(ctx, transform)
				if err != nil {
					return fmt.Errorf("process: failed to initialize output transform: %s - %w", transform.Name, err)
				}
				outputContainer.OutputTransforms = append(outputContainer.OutputTransforms, outputTransform)
			}

			outputs = append(outputs, outputContainer)
		}

		if len(outputs) == 0 {
			return errors.New("process: failed to initialize output, none provided")
		}

		for _, dev := range sensorConfig.Devices {
			deviceImpl, err := device.NewDevice(ctx, dev)
			if err != nil {
				return fmt.Errorf("process: failed to initialize device: %s - %w", dev.Name, err)
			}
			devices = append(devices, deviceImpl)
		}

		sg := &sensorGroup{
			config:          sensorConfig,
			input:           inputImpl,
			inputTransforms: inputTransforms,
			outputGroups:    outputs,
			devices:         devices,
		}

		go s.handleSensor(ctx, sg)

		sensorGroups = append(sensorGroups, sg)
	}

	return nil
}

func (s *service) handleSensor(ctx context.Context, sg *sensorGroup) {
	s.wg.Add(1)
	defer s.wg.Done()

	dataChan, err := sg.input.Subscribe(ctx)
	if err != nil {
		logrus.
			WithError(err).
			WithField("input", reflect.ValueOf(sg.input).Type().String()).
			WithField("sensor", sg.config.Name).
			Error("process: failed to subscribe to input")
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case inputData := <-dataChan:
			if inputData == nil {
				break
			}

			var doNotProcess bool
			for _, transform := range sg.inputTransforms {
				transformData, err := transform.Transform(ctx, inputData)
				if err != nil {
					logrus.WithError(err).Error("process: failed to transform input data")
					continue
				}
				if transformData == nil {
					doNotProcess = true
					break
				}

				inputData.Data = transformData.Data
				for k, v := range transformData.Properties {
					inputData.Properties[k] = v
				}
			}
			if doNotProcess {
				continue
			}

			for _, deviceImpl := range sg.devices {
				deviceData := &device.Data{
					Data:       inputData.Data,
					Properties: make(map[string]interface{}),
				}
				for k, v := range inputData.Properties {
					deviceData.Properties[k] = v
				}

				decodedDeviceData, err := deviceImpl.Decode(ctx, deviceData)
				if err != nil {
					logrus.WithError(err).Error("process: failed to decode device data")
					continue
				}
				if decodedDeviceData == nil {
					continue
				}

				for _, og := range sg.outputGroups {
					outputData := &output.Data{
						Data:       decodedDeviceData.Data,
						Properties: make(map[string]interface{}),
					}
					for k, v := range decodedDeviceData.Properties {
						outputData.Properties[k] = v
					}

					var doNotProcess bool
					for _, transform := range og.OutputTransforms {
						transformData, err := transform.Transform(ctx, outputData)
						if err != nil {
							logrus.WithError(err).Error("process: failed to transform output data")
							continue
						}
						if transformData == nil {
							doNotProcess = true
							break
						}

						outputData.Data = transformData.Data
						for k, v := range transformData.Properties {
							outputData.Properties[k] = v
						}
					}
					if doNotProcess {
						continue
					}

					if err = og.Output.Publish(ctx, outputData); err != nil {
						logrus.WithError(err).Error("process: failed to publish output data")
						continue
					}
				}
			}
		}
	}
}

func (s *service) Close() error {
	s.cancelFunc()
	s.wg.Wait()

	var errs []error
	for _, container := range s.sensorGroups {
		if err := container.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
