package transform

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nikiforov-soft/yasp/config"
)

var (
	ErrTransformNotFound = errors.New("transform not found")
)

var (
	transforms     = make(map[string]Factory)
	transformsLock sync.RWMutex
)

func NewTransform(ctx context.Context, config *config.Transform) (Transform, error) {
	transformsLock.RLock()
	defer transformsLock.RUnlock()

	transformFactory := transforms[strings.ToLower(config.Name)]
	if transformFactory == nil {
		return nil, fmt.Errorf("input %w: %s", ErrTransformNotFound, config.Name)
	}
	return transformFactory(ctx, config)
}

func RegisterTransform(transformName string, transformFactory Factory) error {
	transformsLock.Lock()
	defer transformsLock.Unlock()

	key := strings.ToLower(transformName)
	if _, exists := transforms[key]; exists {
		return fmt.Errorf("transform name: %s already exists", transformName)
	}
	transforms[key] = transformFactory
	return nil
}
