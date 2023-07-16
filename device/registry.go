package device

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nikiforov-soft/yasp/config"
)

var ErrDeviceNotFound = errors.New("device not found")

var (
	devices     = make(map[string]Factory)
	devicesLock sync.RWMutex
)

func NewDevice(ctx context.Context, config *config.Device) (Device, error) {
	devicesLock.RLock()
	defer devicesLock.RUnlock()

	deviceFactory := devices[strings.ToLower(config.Type)]
	if deviceFactory == nil {
		return nil, fmt.Errorf("%w: %s", ErrDeviceNotFound, config.Type)
	}
	return deviceFactory(ctx, config)
}

func RegisterDevice(deviceName string, deviceFactory Factory) error {
	devicesLock.Lock()
	defer devicesLock.Unlock()

	key := strings.ToLower(deviceName)
	if _, exists := devices[key]; exists {
		return fmt.Errorf("device name: %s already exists", deviceName)
	}
	devices[key] = deviceFactory
	return nil
}
