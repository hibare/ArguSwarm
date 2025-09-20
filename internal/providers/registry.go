// Package providers provides a registry for provider implementations.
package providers

import (
	"fmt"

	"github.com/hibare/ArguSwarm/internal/providers/types"
)

// ProviderConstructor is a function type that creates a provider instance.
type ProviderConstructor func() (types.ProviderIface, error)

// ProviderRegistry is a registry for provider constructors.
type ProviderRegistry struct {
	registry map[types.ProviderType]ProviderConstructor
}

// Create creates a provider instance based on the provider type.
func (r *ProviderRegistry) Create(providerType types.ProviderType) (types.ProviderIface, error) {
	constructor, exists := r.registry[providerType]
	if !exists {
		return nil, fmt.Errorf("provider type %s not registered", providerType)
	}
	return constructor()
}

// Register registers a provider constructor for a given provider type.
func (r *ProviderRegistry) Register(providerType types.ProviderType, constructor ProviderConstructor) {
	r.registry[providerType] = constructor
}

// Registry is the global Registry instance.
var Registry = ProviderRegistry{
	registry: make(map[types.ProviderType]ProviderConstructor),
}
