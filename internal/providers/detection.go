// Package providers provides detection and factory functionality for orchestration platforms.
package providers

import (
	"os"
	"strings"

	"github.com/hibare/ArguSwarm/internal/config"
)

// DetectProviderType detects the current orchestration platform.
func DetectProviderType() ProviderType {
	// Check for explicit provider configuration
	providerEnv := strings.ToLower(config.Current.Provider.Type)
	switch providerEnv {
	case "kubernetes", "k8s":
		return ProviderKubernetes
	case "docker-swarm", "swarm":
		return ProviderDockerSwarm
	}

	// Check for Kubernetes environment variables
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return ProviderKubernetes
	}

	// Check for Docker Swarm environment variables
	if os.Getenv("DOCKER_SWARM_MODE") == "1" || os.Getenv("SWARM_MODE") == "1" {
		return ProviderDockerSwarm
	}

	// Check if we're running in a container with swarm labels
	if os.Getenv("DOCKER_SWARM_LABELS") != "" {
		return ProviderDockerSwarm
	}


	// Default to Docker Swarm for backward compatibility
	return ProviderDockerSwarm
}

// NewProviderWithType creates a new provider instance of the specified type.
func NewProviderWithType(providerType ProviderType) (Provider, error) {
	switch providerType {
	case ProviderDockerSwarm:
		return NewDockerSwarmProvider()
	case ProviderKubernetes:
		return NewKubernetesProvider()
	default:
		return nil, ErrUnsupportedProvider
	}
}

// NewProvider creates a new provider instance based on the detected or configured type.
func NewProvider() (Provider, error) {
	providerType := DetectProviderType()
	return NewProviderWithType(providerType)
}
