package input

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/nikiforov-soft/yasp/config"
)

var ErrInputNotFound = errors.New("input not found")

var (
	inputs     = make(map[string]Factory)
	inputsLock sync.RWMutex
)

func NewInput(ctx context.Context, inputName string, config *config.Input) (Input, error) {
	inputsLock.RLock()
	defer inputsLock.RUnlock()

	inputFactory := inputs[strings.ToLower(inputName)]
	if inputFactory == nil {
		return nil, fmt.Errorf("%w: %s", ErrInputNotFound, inputName)
	}
	return inputFactory(ctx, config)
}

func RegisterInput(inputName string, inputFactory Factory) error {
	inputsLock.Lock()
	defer inputsLock.Unlock()

	key := strings.ToLower(inputName)
	if _, exists := inputs[key]; exists {
		return fmt.Errorf("input name: %s already exists", inputName)
	}
	inputs[key] = inputFactory
	return nil
}
