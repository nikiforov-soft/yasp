package process

import (
	"context"
	"errors"

	"github.com/nikiforov-soft/yasp/config"
	"github.com/nikiforov-soft/yasp/device"
	"github.com/nikiforov-soft/yasp/input"
	inputtransform "github.com/nikiforov-soft/yasp/input/transform"
)

type sensorGroup struct {
	config          *config.Sensor
	input           input.Input
	inputTransforms []inputtransform.Transform
	outputs         []outputGroup
	devices         []device.Device
}

func (sg *sensorGroup) Close() error {
	var errs []error
	if err := sg.input.Close(context.Background()); err != nil {
		errs = append(errs, err)
	}
	for _, og := range sg.outputs {
		if err := og.Output.Close(context.Background()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
