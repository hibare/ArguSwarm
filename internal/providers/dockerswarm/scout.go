// Package dockerswarm provides Docker Swarm scout functionality.
package dockerswarm

import (
	"context"
	"log/slog"
	"net/http"

	containerType "github.com/docker/docker/api/types/container"
	imageType "github.com/docker/docker/api/types/image"
	networkType "github.com/docker/docker/api/types/network"
	volumeType "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/hibare/ArguSwarm/internal/node"
	"github.com/hibare/ArguSwarm/internal/providers/errors"
	"github.com/hibare/ArguSwarm/internal/providers/types"
	commonHttp "github.com/hibare/GoCommon/v2/pkg/http"
)

// ScoutAgent implements the scout functionality for Docker Swarm.
type ScoutAgent struct {
	dockerClient *client.Client
	nodeID       string
}

// NewScoutAgent creates a new Docker Swarm scout agent.
func NewScoutAgent() (*ScoutAgent, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &ScoutAgent{
		dockerClient: cli,
		nodeID:       node.GetNodeID(),
	}, nil
}

// GetContainers returns container information from the local Docker daemon.
func (s *ScoutAgent) GetContainers(ctx context.Context) ([]types.ContainerInfo, error) {
	containers, err := s.dockerClient.ContainerList(ctx, containerType.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list containers", "error", err)
		return nil, errors.ErrScoutUnavailable
	}

	result := make([]types.ContainerInfo, 0, len(containers))
	for _, container := range containers {
		// Convert ports
		ports := make([]types.PortInfo, 0, len(container.Ports))
		for _, port := range container.Ports {
			ports = append(ports, types.PortInfo{
				IP:          port.IP,
				PrivatePort: int(port.PrivatePort),
				PublicPort:  int(port.PublicPort),
				Type:        port.Type,
			})
		}

		result = append(result, types.ContainerInfo{
			ID:      container.ID,
			Names:   container.Names,
			Image:   container.Image,
			Status:  container.Status,
			State:   container.State,
			Created: container.Created,
			Ports:   ports,
			Labels:  container.Labels,
			NodeID:  s.nodeID,
		})
	}

	return result, nil
}

// GetImages returns image information from the local Docker daemon.
func (s *ScoutAgent) GetImages(ctx context.Context) ([]types.ImageInfo, error) {
	images, err := s.dockerClient.ImageList(ctx, imageType.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list images", "error", err)
		return nil, errors.ErrScoutUnavailable
	}

	result := make([]types.ImageInfo, 0, len(images))
	for _, image := range images {
		result = append(result, types.ImageInfo{
			ID:          image.ID,
			ParentID:    image.ParentID,
			RepoTags:    image.RepoTags,
			Created:     image.Created,
			Size:        image.Size,
			VirtualSize: image.Size, // Use Size instead of deprecated VirtualSize
			Labels:      image.Labels,
		})
	}

	return result, nil
}

// GetNetworks returns network information from the local Docker daemon.
func (s *ScoutAgent) GetNetworks(ctx context.Context) ([]types.NetworkInfo, error) {
	networks, err := s.dockerClient.NetworkList(ctx, networkType.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list networks", "error", err)
		return nil, errors.ErrScoutUnavailable
	}

	result := make([]types.NetworkInfo, 0, len(networks))
	for _, network := range networks {
		// Convert IPAM config
		var ipam *types.IPAMInfo
		if network.IPAM.Driver != "" || len(network.IPAM.Config) > 0 {
			configs := make([]types.IPAMConfig, 0, len(network.IPAM.Config))
			for _, config := range network.IPAM.Config {
				configs = append(configs, types.IPAMConfig{
					Subnet:  config.Subnet,
					Gateway: config.Gateway,
				})
			}
			ipam = &types.IPAMInfo{
				Driver: network.IPAM.Driver,
				Config: configs,
			}
		}

		result = append(result, types.NetworkInfo{
			Name:       network.Name,
			ID:         network.ID,
			Created:    network.Created.Format("2006-01-02T15:04:05Z07:00"),
			Scope:      network.Scope,
			Driver:     network.Driver,
			EnableIPv6: network.EnableIPv6,
			IPAM:       ipam,
			Labels:     network.Labels,
		})
	}

	return result, nil
}

// GetVolumes returns volume information from the local Docker daemon.
func (s *ScoutAgent) GetVolumes(ctx context.Context) ([]types.VolumeInfo, error) {
	volumes, err := s.dockerClient.VolumeList(ctx, volumeType.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list volumes", "error", err)
		return nil, errors.ErrScoutUnavailable
	}

	result := make([]types.VolumeInfo, 0, len(volumes.Volumes))
	for _, volume := range volumes.Volumes {
		result = append(result, types.VolumeInfo{
			Name:       volume.Name,
			Driver:     volume.Driver,
			Mountpoint: volume.Mountpoint,
			CreatedAt:  volume.CreatedAt,
			Labels:     volume.Labels,
		})
	}

	return result, nil
}

// HandleContainers handles the containers endpoint for Docker Swarm scout.
func (s *ScoutAgent) HandleContainers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	containers, err := s.GetContainers(ctx)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, containers)
}

// HandleImages handles the images endpoint for Docker Swarm scout.
func (s *ScoutAgent) HandleImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	images, err := s.GetImages(ctx)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, images)
}

// HandleNetworks handles the networks endpoint for Docker Swarm scout.
func (s *ScoutAgent) HandleNetworks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	networks, err := s.GetNetworks(ctx)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, networks)
}

// HandleVolumes handles the volumes endpoint for Docker Swarm scout.
func (s *ScoutAgent) HandleVolumes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	volumes, err := s.GetVolumes(ctx)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, volumes)
}
