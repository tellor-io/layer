package combined_handler

import (
	"fmt"
	"sync"
)

var (
	handlerRegistry = make(map[string]CombinedHandler)
	registryMu      sync.RWMutex
)

func RegisterHandler(name string, handler CombinedHandler) {
	registryMu.Lock()
	defer registryMu.Unlock()
	handlerRegistry[name] = handler
}

func GetHandler(name string) (CombinedHandler, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	handler, exists := handlerRegistry[name]
	if !exists {
		return nil, fmt.Errorf("combined handler not found: %s", name)
	}

	return handler, nil
}
