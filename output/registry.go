package output

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nikiforov-soft/yasp/config"
)

var ErrOutputNotFound = errors.New("output not found")

var (
	outputs     = make(map[string]Factory)
	outputsLock sync.RWMutex
)

func NewOutput(ctx context.Context, outputName string, config *config.Output) (Output, error) {
	outputsLock.RLock()
	defer outputsLock.RUnlock()

	outputFactory := outputs[strings.ToLower(outputName)]
	if outputFactory == nil {
		return nil, fmt.Errorf("%w: %s", ErrOutputNotFound, outputName)
	}
	return outputFactory(ctx, config)
}

func RegisterOutput(outputName string, outputFactory Factory) error {
	outputsLock.Lock()
	defer outputsLock.Unlock()

	key := strings.ToLower(outputName)
	if _, exists := outputs[key]; exists {
		return fmt.Errorf("output name: %s already exists", outputName)
	}
	outputs[key] = outputFactory
	return nil
}
