// Package providers provides detection and factory functionality for orchestration platforms.
package providers

import (
	"os"
	"strings"

	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/providers/types"
)

// DetectProviderType detects the current orchestration platform.
func DetectProviderType() types.ProviderType {
	// Check for Kubernetes environment variables
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return types.ProviderKubernetes
	}

	// Check for Docker Swarm environment variables
	if os.Getenv("DOCKER_SWARM_MODE") == "1" || os.Getenv("SWARM_MODE") == "1" {
		return types.ProviderDockerSwarm
	}

	// Check if we're running in a container with swarm labels
	if os.Getenv("DOCKER_SWARM_LABELS") != "" {
		return types.ProviderDockerSwarm
	}

	// Check for explicit provider configuration
	providerEnv := strings.ToLower(config.Current.Provider.Type)
	switch providerEnv {
	case "kubernetes", "k8s":
		return types.ProviderKubernetes
	case "docker-swarm", "swarm":
		return types.ProviderDockerSwarm
	}

	// Default to Docker Swarm for backward compatibility
	return types.ProviderDockerSwarm
}

// NewProvider creates a new provider instance based on the detected or configured type.
func NewProvider() (types.ProviderIface, error) {
	providerType := DetectProviderType()
	return Registry.Create(providerType)
}
