package sensor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nikiforov-soft/yasp/config"
)

var ErrSensorNotFound = errors.New("sensor not found")

var (
	sensors     = make(map[string]Factory)
	sensorsLock sync.RWMutex
)

func NewSensor(ctx context.Context, config *config.Device) (Sensor, error) {
	sensorsLock.RLock()
	defer sensorsLock.RUnlock()

	sensorFactory := sensors[strings.ToLower(config.Type)]
	if sensorFactory == nil {
		return nil, fmt.Errorf("%w: %s", ErrSensorNotFound, config.Type)
	}
	return sensorFactory(ctx, config)
}

func RegisterSensor(sensorName string, sensorFactory Factory) error {
	sensorsLock.Lock()
	defer sensorsLock.Unlock()

	key := strings.ToLower(sensorName)
	if _, exists := sensors[key]; exists {
		return fmt.Errorf("sensor name: %s already exists", sensorName)
	}
	sensors[key] = sensorFactory
	return nil
}
