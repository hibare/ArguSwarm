// Package providers provides factory functions for creating provider instances.
package providers

import (
	"errors"
)

// NewDockerSwarmProvider creates a new Docker Swarm provider.
// This function is implemented in the dockerswarm package to avoid import cycles.
func NewDockerSwarmProvider() (Provider, error) {
	// This will be implemented by importing the dockerswarm package
	// when the overseer/scout packages are initialized
	return nil, errors.New("docker Swarm provider not available - import cycle prevention")
}

// NewKubernetesProvider creates a new Kubernetes provider.
// This function is implemented in the kubernetes package to avoid import cycles.
func NewKubernetesProvider() (Provider, error) {
	// This will be implemented by importing the kubernetes package
	// when the overseer/scout packages are initialized
	return nil, errors.New("kubernetes provider not available - import cycle prevention")
}
