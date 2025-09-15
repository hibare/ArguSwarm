// Package providers defines the interface and types for orchestration platform providers.
package providers

import (
	"context"
	"errors"
)

// ProviderType represents the type of orchestration platform.
type ProviderType string

const (
	// ProviderDockerSwarm represents Docker Swarm orchestration.
	ProviderDockerSwarm ProviderType = "docker-swarm"
	// ProviderKubernetes represents Kubernetes orchestration.
	ProviderKubernetes ProviderType = "kubernetes"
)

// ContainerInfo represents container information across different providers.
type ContainerInfo struct {
	ID      string            `json:"id"`
	Names   []string          `json:"names"`
	Image   string            `json:"image"`
	Status  string            `json:"status"`
	State   string            `json:"state"`
	Created int64             `json:"created"`
	Ports   []PortInfo        `json:"ports"`
	Labels  map[string]string `json:"labels"`
	NodeID  string            `json:"node_id,omitempty"`
}

// PortInfo represents port information for containers.
type PortInfo struct {
	IP          string `json:"ip"`
	PrivatePort int    `json:"private_port"`
	PublicPort  int    `json:"public_port"`
	Type        string `json:"type"`
}

// ImageInfo represents image information across different providers.
type ImageInfo struct {
	ID          string            `json:"id"`
	ParentID    string            `json:"parent_id,omitempty"`
	RepoTags    []string          `json:"repo_tags"`
	Created     int64             `json:"created"`
	Size        int64             `json:"size"`
	VirtualSize int64             `json:"virtual_size,omitempty"`
	Labels      map[string]string `json:"labels"`
}

// NetworkInfo represents network information across different providers.
type NetworkInfo struct {
	Name       string            `json:"name"`
	ID         string            `json:"id"`
	Created    string            `json:"created"`
	Scope      string            `json:"scope"`
	Driver     string            `json:"driver"`
	EnableIPv6 bool              `json:"enable_ipv6"`
	IPAM       *IPAMInfo         `json:"ipam,omitempty"`
	Labels     map[string]string `json:"labels"`
}

// IPAMInfo represents IPAM configuration for networks.
type IPAMInfo struct {
	Driver string       `json:"driver"`
	Config []IPAMConfig `json:"config"`
}

// IPAMConfig represents IPAM configuration details.
type IPAMConfig struct {
	Subnet  string `json:"subnet"`
	Gateway string `json:"gateway"`
}

// VolumeInfo represents volume information across different providers.
type VolumeInfo struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	CreatedAt  string            `json:"created_at"`
	Labels     map[string]string `json:"labels"`
}

// ScoutInfo represents scout/node information.
type ScoutInfo struct {
	NodeID   string `json:"node_id"`
	Address  string `json:"address"`
	Status   string `json:"status"`
	LastSeen string `json:"last_seen,omitempty"`
}

// Provider defines the interface that all orchestration providers must implement.
type Provider interface {
	// GetType returns the provider type.
	GetType() ProviderType

	// GetScouts returns a list of available scouts/nodes.
	GetScouts(ctx context.Context) ([]ScoutInfo, error)

	// GetContainers returns container information from all scouts.
	GetContainers(ctx context.Context) ([]ContainerInfo, error)

	// GetImages returns image information from all scouts.
	GetImages(ctx context.Context) ([]ImageInfo, error)

	// GetNetworks returns network information from all scouts.
	GetNetworks(ctx context.Context) ([]NetworkInfo, error)

	// GetVolumes returns volume information from all scouts.
	GetVolumes(ctx context.Context) ([]VolumeInfo, error)

	// CheckContainerHealth checks if a container is healthy.
	CheckContainerHealth(ctx context.Context, name string, filter string) (map[string]bool, error)
}

// ProviderFactory creates provider instances based on configuration.
type ProviderFactory struct{}

// NewProvider creates a new provider instance based on the current configuration.
func (f *ProviderFactory) NewProvider() (Provider, error) {
	// This will be implemented to detect the environment and return the appropriate provider
	// For now, we'll implement this in the main provider file
	return nil, errors.New("provider factory not implemented")
}
