// Package kubernetes provides Kubernetes orchestration platform support.
package kubernetes

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hibare/ArguSwarm/internal/providers/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// ContainerStateRunning indicates a container is currently running.
	ContainerStateRunning = "running"

	// ContainerStateHealthy indicates a container is in a healthy state.
	ContainerStateHealthy = "healthy"

	// ContainerStateTerminated indicates a container is terminated.
	ContainerStateTerminated = "terminated"

	// ContainerStateWaiting indicates a container is waiting.
	ContainerStateWaiting = "waiting"

	// ContainerStateUnknown indicates a container is unknown.
	ContainerStateUnknown = "unknown"
)

// Provider implements the types.Provider interface for Kubernetes.
type Provider struct {
	clientset *kubernetes.Clientset
}

// NewProvider creates a new Kubernetes provider instance.
func NewProvider() (types.ProviderIface, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &Provider{
		clientset: clientset,
	}, nil
}

// GetType returns the provider type.
func (p *Provider) GetType() types.ProviderType {
	return types.ProviderKubernetes
}

// GetScouts returns a list of available nodes in the Kubernetes cluster.
// Note: For Kubernetes, scouts are not needed as we query the API directly.
func (p *Provider) GetScouts(_ context.Context) ([]types.ScoutInfo, error) {
	// For Kubernetes, we don't use scouts - we query the API directly
	// Return empty list to indicate no scouts are needed
	return []types.ScoutInfo{}, nil
}

// GetContainers returns pod information from the Kubernetes cluster.
func (p *Provider) GetContainers(ctx context.Context) ([]types.ContainerInfo, error) {
	pods, err := p.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	containers := make([]types.ContainerInfo, 0)
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			// Get container status
			status := ContainerStateUnknown
			state := ContainerStateUnknown
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Name == container.Name {
					switch {
					case containerStatus.State.Running != nil:
						status = ContainerStateRunning
						state = ContainerStateRunning
					case containerStatus.State.Waiting != nil:
						status = ContainerStateWaiting
						state = ContainerStateWaiting
					case containerStatus.State.Terminated != nil:
						status = ContainerStateTerminated
						state = ContainerStateTerminated
					}
					break
				}
			}

			// Convert ports
			ports := make([]types.PortInfo, 0, len(container.Ports))
			for _, port := range container.Ports {
				ports = append(ports, types.PortInfo{
					IP:          port.HostIP,
					PrivatePort: int(port.ContainerPort),
					PublicPort:  int(port.HostPort),
					Type:        string(port.Protocol),
				})
			}

			containers = append(containers, types.ContainerInfo{
				ID:      fmt.Sprintf("%s/%s", pod.Namespace, container.Name),
				Names:   []string{fmt.Sprintf("%s/%s", pod.Name, container.Name)},
				Image:   container.Image,
				Status:  status,
				State:   state,
				Created: pod.CreationTimestamp.Unix(),
				Ports:   ports,
				Labels:  pod.Labels,
				NodeID:  pod.Spec.NodeName,
			})
		}
	}

	return containers, nil
}

// GetImages returns image information from the Kubernetes cluster.
func (p *Provider) GetImages(ctx context.Context) ([]types.ImageInfo, error) {
	pods, err := p.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	imageMap := make(map[string]types.ImageInfo)
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			imageID := container.Image
			if _, exists := imageMap[imageID]; !exists {
				imageMap[imageID] = types.ImageInfo{
					ID:       imageID,
					RepoTags: []string{container.Image},
					Labels:   make(map[string]string),
				}
			}
		}
	}

	images := make([]types.ImageInfo, 0, len(imageMap))
	for _, image := range imageMap {
		images = append(images, image)
	}

	return images, nil
}

// GetNetworks returns network information from the Kubernetes cluster.
func (p *Provider) GetNetworks(ctx context.Context) ([]types.NetworkInfo, error) {
	services, err := p.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	networks := make([]types.NetworkInfo, 0, len(services.Items))
	for _, service := range services.Items {
		// Convert service ports to network info
		ports := make([]types.IPAMConfig, 0, len(service.Spec.Ports))
		for _, port := range service.Spec.Ports {
			ports = append(ports, types.IPAMConfig{
				Subnet:  strconv.Itoa(int(port.Port)),
				Gateway: service.Spec.ClusterIP,
			})
		}

		networks = append(networks, types.NetworkInfo{
			Name:       service.Name,
			ID:         string(service.UID),
			Created:    service.CreationTimestamp.Format(time.RFC3339),
			Scope:      "cluster",
			Driver:     "kube-proxy",
			EnableIPv6: false,
			IPAM: &types.IPAMInfo{
				Driver: "kubernetes",
				Config: ports,
			},
			Labels: service.Labels,
		})
	}

	return networks, nil
}

// GetVolumes returns volume information from the Kubernetes cluster.
func (p *Provider) GetVolumes(ctx context.Context) ([]types.VolumeInfo, error) {
	pvcs, err := p.clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list persistent volume claims: %w", err)
	}

	volumes := make([]types.VolumeInfo, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		volumes = append(volumes, types.VolumeInfo{
			Name:       pvc.Name,
			Driver:     "kubernetes.io/pvc",
			Mountpoint: fmt.Sprintf("/var/lib/kubelet/pods/%s/volumes/kubernetes.io~pvc/%s", pvc.UID, pvc.Name),
			CreatedAt:  pvc.CreationTimestamp.Format(time.RFC3339),
			Labels:     pvc.Labels,
		})
	}

	return volumes, nil
}

// CheckContainerHealth checks if a pod/container is healthy.
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
