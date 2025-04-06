package scout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	containerType "github.com/docker/docker/api/types/container"
	imageType "github.com/docker/docker/api/types/image"
	networkType "github.com/docker/docker/api/types/network"
	volumeType "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/constants"
	"github.com/hibare/ArguSwarm/internal/node"
	commonHttp "github.com/hibare/GoCommon/v2/pkg/http"
)

// Agent represents a network scout agent.
type Agent struct {
	dockerClient *client.Client
	overseerAddr string
	nodeID       string
	httpClient   *http.Client
	context      context.Context
}

// HealthStatus represents the health status of a scout.
type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	NodeID    string    `json:"node_id"`
}

type nodeIDKey struct{}

// NewAgent creates a new Agent instance.
func NewAgent() (*Agent, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	nodeID := node.GetNodeID()

	// Create context and add node ID
	ctx := context.Background()

	ctx = context.WithValue(ctx, nodeIDKey{}, nodeID)

	return &Agent{
		dockerClient: cli,
		overseerAddr: config.Current.Scout.OverseerServerAddress,
		nodeID:       nodeID,
		httpClient:   &http.Client{Timeout: config.Current.HTTPClient.Timeout},
		context:      ctx,
	}, nil
}

// Start begins the agent's operation.
func (s *Agent) Start() error {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.NoCache)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(constants.DefaultServerTimeout))
	router.Use(middleware.StripSlashes)
	router.Use(middleware.CleanPath)
	router.Use(middleware.Heartbeat(constants.PingPath))

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/containers", s.handleContainers)
		r.Get("/images", s.handleImages)
		r.Get("/networks", s.handleNetworks)
		r.Get("/volumes", s.handlerVolumes)
	})

	srvAddr := fmt.Sprintf(":%d", constants.DefaultScoutPort)

	srv := &http.Server{
		Handler:      router,
		Addr:         srvAddr,
		WriteTimeout: config.Current.Server.WriteTimeout,
		ReadTimeout:  config.Current.Server.ReadTimeout,
		IdleTimeout:  config.Current.Server.IdleTimeout,
	}

	slog.InfoContext(s.context, "Scout agent started", "address", srvAddr)

	// Start health check routine
	go s.pingOverseer()

	// Run our server in a goroutine so that it doesn't block.
	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(s.context, "failed to start server", "error", err)
			errChan <- err
		}
	}()

	// check if errChan has any error
	select {
	case err := <-errChan:
		return err
	default:
	}

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(s.context, constants.DefaultServerShutdownGracePeriod)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.ErrorContext(s.context, "Server shutdown failed", "error", err)
		return err
	}

	slog.InfoContext(s.context, "Server shutdown successfully")

	return nil
}

func (s *Agent) pingOverseer() {
	// Try initial connection up to 3 times
	maxAttempts := 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			slog.InfoContext(
				s.context,
				"Retrying initial health check",
				"attempt", attempt,
				"max_attempts", maxAttempts,
			)
			time.Sleep(constants.DefaultRetryInterval)
		}

		if err := s.sendHealthCheck(); err == nil {
			break
		}

		if attempt == maxAttempts {
			slog.ErrorContext(s.context, "Failed to establish initial connection to overseer after all retries")
		}
	}

	// Start periodic health checks
	ticker := time.NewTicker(constants.DefaultHealthCheckInterval)
	slog.DebugContext(
		s.context,
		"Starting health check routine",
		"node_id", s.nodeID,
		"overseer_addr", s.overseerAddr,
		"interval", constants.DefaultHealthCheckInterval,
	)

	for range ticker.C {
		_ = s.sendHealthCheck()
	}
}

// Helper function to send health check.
func (s *Agent) sendHealthCheck() error {
	slog.DebugContext(s.context, "Pinging overseer", "node_id", s.nodeID)
	status := HealthStatus{
		Timestamp: time.Now(),
		NodeID:    s.nodeID,
	}

	jsonData, err := json.Marshal(status)
	if err != nil {
		slog.ErrorContext(s.context, "Error marshaling health status", "error", err)
		return err
	}
	req, err := http.NewRequestWithContext(
		s.context,
		http.MethodPost,
		fmt.Sprintf("%s/api/v1/scouts/ping", s.overseerAddr),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		slog.ErrorContext(s.context, "Error sending health check", "error", err)
		return err
	}
	if closeErr := resp.Body.Close(); closeErr != nil {
		return closeErr
	}
	slog.DebugContext(s.context, "Pinged overseer", "node_id", s.nodeID, "status", resp.StatusCode)
	if resp.StatusCode == http.StatusOK {
		slog.InfoContext(s.context, "Scout registered", "node_id", s.nodeID)
	} else {
		slog.ErrorContext(s.context, "Unexpected status code", "status_code", resp.StatusCode)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (s *Agent) handleContainers(w http.ResponseWriter, _ *http.Request) {
	ctx := context.Background()
	containers, err := s.dockerClient.ContainerList(ctx, containerType.ListOptions{})
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, containers)
}

func (s *Agent) handleImages(w http.ResponseWriter, _ *http.Request) {
	ctx := context.Background()
	images, err := s.dockerClient.ImageList(ctx, imageType.ListOptions{})
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, images)
}

func (s *Agent) handleNetworks(w http.ResponseWriter, _ *http.Request) {
	ctx := context.Background()
	networks, err := s.dockerClient.NetworkList(ctx, networkType.ListOptions{})
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, networks)
}

func (s *Agent) handlerVolumes(w http.ResponseWriter, _ *http.Request) {
	ctx := context.Background()
	volumes, err := s.dockerClient.VolumeList(ctx, volumeType.ListOptions{})
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, volumes.Volumes)
}
