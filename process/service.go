package process

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/input"
	inputtransform "github.com/nikiforov-soft/yasp/input/transform"
	"github.com/nikiforov-soft/yasp/output"
	outputtransform "github.com/nikiforov-soft/yasp/output/transform"
	"github.com/nikiforov-soft/yasp/sensor"
	"github.com/sirupsen/logrus"
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
		var sensors []sensor.Sensor
		var err error
		if mqtt := sensorConfig.Input.Mqtt; mqtt != nil {
			inputImpl, err = input.NewInput(ctx, "mqtt", mqtt)
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
			if mqtt := o.Mqtt; mqtt != nil {
				outputImpl, err := output.NewOutput(ctx, "mqtt", mqtt)
				if err != nil {
					return fmt.Errorf("process: failed to initialize output: %w", err)
				}
				outputContainer.Output = outputImpl
			}

			if outputContainer.Output == nil {
				return errors.New("process: failed to initialize output, none provided")
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

		for _, device := range sensorConfig.Devices {
			sensorImpl, err := sensor.NewSensor(ctx, device)
			if err != nil {
				return fmt.Errorf("process: failed to initialize sensor: %s - %w", device.Name, err)
			}
			sensors = append(sensors, sensorImpl)
		}

		sg := &sensorGroup{
			config:          sensorConfig,
			input:           inputImpl,
			inputTransforms: inputTransforms,
			outputs:         outputs,
			sensors:         sensors,
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

			for _, transform := range sg.inputTransforms {
				data, err := transform.Transform(ctx, inputData.Data)
				if err != nil {
					logrus.WithError(err).Error("process: failed to transform input data")
					continue
				}
				inputData.Data = data
			}

			for _, sensorImpl := range sg.sensors {
				decodedData, err := sensorImpl.Decode(ctx, sg.config, inputData.Data)
				if err != nil {
					logrus.WithError(err).Error("process: failed to decode data: %w", err)
					continue
				}
				if decodedData == nil {
					continue
				}

				for _, og := range sg.outputs {
					outputData := decodedData.Data
					for _, transform := range og.OutputTransforms {
						data, err := transform.Transform(ctx, outputData)
						if err != nil {
							logrus.WithError(err).Error("process: failed to transform input data: %w", err)
							continue
						}
						outputData = data
					}

					err = og.Output.Publish(ctx, &output.Data{
						Data:       outputData,
						Properties: decodedData.Properties,
					})
					if err != nil {
						logrus.WithError(err).Error("process: failed to publish output data: %w", err)
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
