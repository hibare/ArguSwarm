// Package dockerswarm provides Docker Swarm orchestration platform support.
package dockerswarm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/providers"
	commonConcurrency "github.com/hibare/GoCommon/v2/pkg/concurrency"
	commonMiddleware "github.com/hibare/GoCommon/v2/pkg/http/middleware"
)

// Container state constants.
const (
	// ContainerStateRunning indicates a container is currently running.
	ContainerStateRunning = "running"

	// ContainerStateHealthy indicates a container is in a healthy state.
	ContainerStateHealthy = "healthy"
)

// Provider implements the providers.Provider interface for Docker Swarm.
type Provider struct {
	httpClient *http.Client
}

// NewProvider creates a new Docker Swarm provider instance.
func NewProvider() (providers.Provider, error) {
	return &Provider{
		httpClient: &http.Client{Timeout: config.Current.HTTPClient.Timeout},
	}, nil
}

// GetType returns the provider type.
func (p *Provider) GetType() providers.ProviderType {
	return providers.ProviderDockerSwarm
}

// GetScouts returns a list of available scouts in the Docker Swarm.
func (p *Provider) GetScouts(ctx context.Context) ([]providers.ScoutInfo, error) {
	resolver := &net.Resolver{}
	ips, err := resolver.LookupIPAddr(ctx, "tasks.scout")
	if err != nil {
		return nil, err
	}

	scouts := make([]providers.ScoutInfo, len(ips))
	for i, ipAddr := range ips {
		scouts[i] = providers.ScoutInfo{
			NodeID:  ipAddr.IP.String(),
			Address: ipAddr.IP.String(),
			Status:  "active",
		}
	}

	slog.DebugContext(ctx, "Docker Swarm scouts", "scouts", scouts, "count", len(scouts))
	return scouts, nil
}

// GetContainers returns container information from all scouts.
func (p *Provider) GetContainers(ctx context.Context) ([]providers.ContainerInfo, error) {
	results, err := p.queryScouts(ctx, "containers")
	if err != nil {
		return nil, err
	}

	containers := make([]providers.ContainerInfo, 0, len(results))
	for _, raw := range results {
		// Convert map to JSON and then to container.Summary struct
		jsonData, jsonErr := json.Marshal(raw)
		if jsonErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal container data", "error", jsonErr)
			continue
		}

		var containerSummary container.Summary
		if unmarshalErr := json.Unmarshal(jsonData, &containerSummary); unmarshalErr != nil {
			slog.ErrorContext(ctx, "Failed to unmarshal container data", "error", unmarshalErr)
			continue
		}

		// Convert to our provider format
		containerInfo := providers.ContainerInfo{
			ID:      containerSummary.ID,
			Names:   containerSummary.Names,
			Image:   containerSummary.Image,
			Status:  containerSummary.Status,
			State:   containerSummary.State,
			Created: containerSummary.Created,
			Labels:  containerSummary.Labels,
		}

		// Convert ports
		for _, port := range containerSummary.Ports {
			containerInfo.Ports = append(containerInfo.Ports, providers.PortInfo{
				IP:          port.IP,
				PrivatePort: int(port.PrivatePort),
				PublicPort:  int(port.PublicPort),
				Type:        port.Type,
			})
		}

		containers = append(containers, containerInfo)
	}

	return containers, nil
}

// GetImages returns image information from all scouts.
func (p *Provider) GetImages(ctx context.Context) ([]providers.ImageInfo, error) {
	results, err := p.queryScouts(ctx, "images")
	if err != nil {
		return nil, err
	}

	images := make([]providers.ImageInfo, 0, len(results))
	for _, raw := range results {
		jsonData, jsonErr := json.Marshal(raw)
		if jsonErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal image data", "error", jsonErr)
			continue
		}

		var imageSummary struct {
			ID          string            `json:"Id"`
			ParentID    string            `json:"ParentId"`
			RepoTags    []string          `json:"RepoTags"`
			Created     int64             `json:"Created"`
			Size        int64             `json:"Size"`
			VirtualSize int64             `json:"VirtualSize"`
			Labels      map[string]string `json:"Labels"`
		}

		if unmarshalErr := json.Unmarshal(jsonData, &imageSummary); unmarshalErr != nil {
			slog.ErrorContext(ctx, "Failed to unmarshal image data", "error", unmarshalErr)
			continue
		}

		images = append(images, providers.ImageInfo{
			ID:          imageSummary.ID,
			ParentID:    imageSummary.ParentID,
			RepoTags:    imageSummary.RepoTags,
			Created:     imageSummary.Created,
			Size:        imageSummary.Size,
			VirtualSize: imageSummary.VirtualSize,
			Labels:      imageSummary.Labels,
		})
	}

	return images, nil
}

// GetNetworks returns network information from all scouts.
func (p *Provider) GetNetworks(ctx context.Context) ([]providers.NetworkInfo, error) {
	results, err := p.queryScouts(ctx, "networks")
	if err != nil {
		return nil, err
	}

	networks := make([]providers.NetworkInfo, 0, len(results))
	for _, raw := range results {
		jsonData, jsonErr := json.Marshal(raw)
		if jsonErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal network data", "error", jsonErr)
			continue
		}

		var networkSummary struct {
			Name       string              `json:"Name"`
			ID         string              `json:"Id"`
			Created    string              `json:"Created"`
			Scope      string              `json:"Scope"`
			Driver     string              `json:"Driver"`
			EnableIPv6 bool                `json:"EnableIPv6"`
			IPAM       *providers.IPAMInfo `json:"IPAM"`
			Labels     map[string]string   `json:"Labels"`
		}

		if unmarshalErr := json.Unmarshal(jsonData, &networkSummary); unmarshalErr != nil {
			slog.ErrorContext(ctx, "Failed to unmarshal network data", "error", unmarshalErr)
			continue
		}

		networks = append(networks, providers.NetworkInfo{
			Name:       networkSummary.Name,
			ID:         networkSummary.ID,
			Created:    networkSummary.Created,
			Scope:      networkSummary.Scope,
			Driver:     networkSummary.Driver,
			EnableIPv6: networkSummary.EnableIPv6,
			IPAM:       networkSummary.IPAM,
			Labels:     networkSummary.Labels,
		})
	}

	return networks, nil
}

// GetVolumes returns volume information from all scouts.
func (p *Provider) GetVolumes(ctx context.Context) ([]providers.VolumeInfo, error) {
	results, err := p.queryScouts(ctx, "volumes")
	if err != nil {
		return nil, err
	}

	volumes := make([]providers.VolumeInfo, 0, len(results))
	for _, raw := range results {
		jsonData, jsonErr := json.Marshal(raw)
		if jsonErr != nil {
			slog.ErrorContext(ctx, "Failed to marshal volume data", "error", jsonErr)
			continue
		}

		var volumeSummary struct {
			Name       string            `json:"Name"`
			Driver     string            `json:"Driver"`
			Mountpoint string            `json:"Mountpoint"`
			CreatedAt  string            `json:"CreatedAt"`
			Labels     map[string]string `json:"Labels"`
		}

		if unmarshalErr := json.Unmarshal(jsonData, &volumeSummary); unmarshalErr != nil {
			slog.ErrorContext(ctx, "Failed to unmarshal volume data", "error", unmarshalErr)
			continue
		}

		volumes = append(volumes, providers.VolumeInfo{
			Name:       volumeSummary.Name,
			Driver:     volumeSummary.Driver,
			Mountpoint: volumeSummary.Mountpoint,
			CreatedAt:  volumeSummary.CreatedAt,
			Labels:     volumeSummary.Labels,
		})
	}

	return volumes, nil
}

// CheckContainerHealth checks if a container is healthy.
func (p *Provider) CheckContainerHealth(ctx context.Context, name string, filter string) (map[string]bool, error) {
	containers, err := p.GetContainers(ctx)
	if err != nil {
		return nil, err
	}

	containerState := map[string]bool{}

	// Define the matches map
	matches := map[string]func(string, string) bool{
		"starts_with": strings.HasPrefix,
		"ends_with":   strings.HasSuffix,
		"contains":    strings.Contains,
		"equal":       func(s1, s2 string) bool { return s1 == s2 },
	}

	// Check if the container name matches the filter
	for _, container := range containers {
		for _, cName := range container.Names {
			cName = strings.TrimPrefix(cName, "/")
			if matchFunc, filterOk := matches[filter]; filterOk && matchFunc(cName, name) {
				state := container.State == ContainerStateRunning ||
					container.State == ContainerStateHealthy

				containerState[cName] = state
			}
		}
	}

	return containerState, nil
}

// queryScouts queries all scouts for a specific resource type.
func (p *Provider) queryScouts(ctx context.Context, resource string) ([]any, error) {
	scouts, err := p.GetScouts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get scouts: %w", err)
	}

	if len(scouts) == 0 {
		return []any{}, nil
	}

	// Create tasks for parallel execution
	resultsMap := sync.Map{}
	tasks := make([]commonConcurrency.ParallelTask, len(scouts))

	for i, s := range scouts {
		scoutIP := s.Address // capture range variable
		tasks[i] = commonConcurrency.ParallelTask{
			Name: fmt.Sprintf("Scout-%s", scoutIP),
			Task: func(ctx context.Context) error {
				result, rErr := p.fetchResource(ctx, scoutIP, resource)
				if rErr != nil {
					return fmt.Errorf("scout %s failed: %w", scoutIP, rErr)
				}
				resultsMap.Store(scoutIP, result)
				return nil
			},
		}
		slog.DebugContext(ctx, "Scout query created", "scout", scoutIP)
	}

	slog.DebugContext(ctx, "Executing scout queries", "count", len(tasks), "concurrency", config.Current.Overseer.MaxConcurrentScoutsQuery)
	// Execute tasks with worker pool
	errsMap := commonConcurrency.RunParallelTasks(
		ctx,
		commonConcurrency.ParallelOptions{
			WorkerCount: config.Current.Overseer.MaxConcurrentScoutsQuery,
		},
		tasks...)

	// Handle errors
	if len(errsMap) == len(scouts) {
		return nil, fmt.Errorf("all scout queries failed: %v", errsMap)
	}

	if len(errsMap) > 0 {
		for task, err := range errsMap {
			slog.ErrorContext(ctx, "Scout query failed", "task", task, "error", err)
		}
	}

	// Collect results
	results := make([]any, 0)
	resultsMap.Range(func(_, value interface{}) bool {
		if v, ok := value.([]any); ok {
			results = append(results, v...)
		}
		return true
	})

	return results, nil
}

// fetchResource fetches a specific resource from a scout.
func (p *Provider) fetchResource(ctx context.Context, scoutIP string, resourceType string) ([]any, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(scoutIP, strconv.Itoa(config.Current.Scout.Port)),
		Path:   "/api/v1/" + resourceType,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create request", "error", err)
		return nil, providers.ErrScoutUnavailable
	}

	req.Header.Set(commonMiddleware.AuthHeaderName, config.Current.Server.SharedSecret)
	req.Header.Set("User-Agent", "ArguSwarm-Overseer/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch resource", "error", err)
		return nil, providers.ErrScoutUnavailable
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "Failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Unexpected status code", "status", resp.StatusCode)
		return nil, providers.ErrScoutUnavailable
	}

	var result []any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		slog.ErrorContext(ctx, "Failed to decode response", "error", decodeErr)
		return nil, providers.ErrScoutUnavailable
	}

	return result, nil
}
