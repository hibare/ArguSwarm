package scout

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	containerType "github.com/docker/docker/api/types/container"
	imageType "github.com/docker/docker/api/types/image"
	networkType "github.com/docker/docker/api/types/network"
	volumeType "github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/constants"
	"github.com/hibare/ArguSwarm/internal/middleware/security"
	"github.com/hibare/ArguSwarm/internal/node"
	commonHttp "github.com/hibare/GoCommon/v2/pkg/http"
	commonMiddleware "github.com/hibare/GoCommon/v2/pkg/http/middleware"
)

var (
	// ErrFailedToRetrieveImageInformation is an error that occurs when a resource cannot be retrieved.
	ErrFailedToRetrieveImageInformation = errors.New("failed to retrieve image information")

	// ErrFailedToRetrieveContainerInformation is an error that occurs when a resource cannot be retrieved.
	ErrFailedToRetrieveContainerInformation = errors.New("failed to retrieve container information")

	// ErrFailedToRetrieveNetworkInformation is an error that occurs when a resource cannot be retrieved.
	ErrFailedToRetrieveNetworkInformation = errors.New("failed to retrieve network information")

	// ErrFailedToRetrieveVolumeInformation is an error that occurs when a resource cannot be retrieved.
	ErrFailedToRetrieveVolumeInformation = errors.New("failed to retrieve volume information")
)

// Agent represents a network scout agent.
type Agent struct {
	dockerClient *client.Client
	nodeID       string
	httpClient   *http.Client
}

// NewAgent creates a new Agent instance with secure Docker client.
func NewAgent() (*Agent, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	nodeID := node.GetNodeID()

	return &Agent{
		dockerClient: cli,
		nodeID:       nodeID,
		httpClient:   &http.Client{Timeout: config.Current.HTTPClient.Timeout},
	}, nil
}

// Start begins the agent's operation with enhanced security.
func (s *Agent) Start() error {
	ctx := context.Background()
	router := chi.NewRouter()

	// Basic middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.NoCache)
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(constants.DefaultServerTimeout))
	router.Use(middleware.StripSlashes)
	router.Use(middleware.CleanPath)
	router.Use(middleware.Heartbeat(constants.PingPath))

	// Use common security middleware
	router.Use(security.BasicSecurity)

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return commonMiddleware.TokenAuth(next, []string{config.Current.Server.SharedSecret})
		})
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

	slog.InfoContext(ctx, "Scout agent started", "address", srvAddr)

	// Run our server in a goroutine so that it doesn't block.
	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "failed to start server", "error", err)
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
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultServerShutdownGracePeriod)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "Server shutdown failed", "error", err)
		return err
	}

	slog.InfoContext(ctx, "Server shutdown successfully")

	return nil
}

func (s *Agent) handleContainers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	containers, err := s.dockerClient.ContainerList(ctx, containerType.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list containers", "error", err)
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError,
			ErrFailedToRetrieveContainerInformation)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, containers)
}

func (s *Agent) handleImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	images, err := s.dockerClient.ImageList(ctx, imageType.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list images", "error", err)
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError,
			ErrFailedToRetrieveImageInformation)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, images)
}

func (s *Agent) handleNetworks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	networks, err := s.dockerClient.NetworkList(ctx, networkType.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list networks", "error", err)
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError,
			ErrFailedToRetrieveNetworkInformation)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, networks)
}

func (s *Agent) handlerVolumes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	volumes, err := s.dockerClient.VolumeList(ctx, volumeType.ListOptions{})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to list volumes", "error", err)
		commonHttp.WriteErrorResponse(w, http.StatusInternalServerError,
			ErrFailedToRetrieveVolumeInformation)
		return
	}
	commonHttp.WriteJsonResponse(w, http.StatusOK, volumes.Volumes)
}
