package overseer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"github.com/ggicci/httpin"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/hibare/ArguSwarm/internal/assets"
	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/constants"
	"github.com/hibare/ArguSwarm/internal/middleware/security"
	"github.com/hibare/ArguSwarm/internal/providers"
	dockerSwarm "github.com/hibare/ArguSwarm/internal/providers/dockerswarm"
	"github.com/hibare/ArguSwarm/internal/providers/kubernetes"
	commonHttp "github.com/hibare/GoCommon/v2/pkg/http"
	commonMiddleware "github.com/hibare/GoCommon/v2/pkg/http/middleware"
)

var (
	// ErrFailedToCreateRequest is an error that occurs when a request cannot be created.
	ErrFailedToCreateRequest = errors.New("failed to create request")

	// ErrFailedToFetchResource is an error that occurs when a resource cannot be fetched.
	ErrFailedToFetchResource = errors.New("failed to fetch resource")

	// ErrFailedToDecodeResponse is an error that occurs when a response cannot be decoded.
	ErrFailedToDecodeResponse = errors.New("failed to decode response")

	// ErrUnexpectedStatusCode is an error that occurs when an unexpected status code is received.
	ErrUnexpectedStatusCode = errors.New("unexpected status code")

	// ErrInvalidRequestFormat is an error that occurs when a request format is invalid.
	ErrInvalidRequestFormat = errors.New("invalid request format")

	// ErrInvalidRemoteAddress is an error that occurs when a remote address is invalid.
	ErrInvalidRemoteAddress = errors.New("invalid remote address")
)

const (
	// FilterStartsWith checks if the string starts with the given prefix.
	FilterStartsWith = "starts_with"

	// FilterEndsWith checks if the string ends with the given suffix.
	FilterEndsWith = "ends_with"

	// FilterContains checks if the string contains the given substring.
	FilterContains = "contains"

	// FilterEqual checks if the string is equal to the given value.
	FilterEqual = "equal"
)

// Overseer manages the health and status of scout agents.
type Overseer struct {
	provider providers.Provider
}

// NewOverseer creates a new Overseer instance.
func NewOverseer() (*Overseer, error) {
	providerType := providers.DetectProviderType()

	var (
		provider providers.Provider
		err      error
	)

	switch providerType {
	case providers.ProviderDockerSwarm:
		provider, err = dockerSwarm.NewProvider()
	case providers.ProviderKubernetes:
		provider, err = kubernetes.NewProvider()
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &Overseer{
		provider: provider,
	}, nil
}

type containerHealthyInput struct {
	Name   string `in:"path=name"`
	Filter string `in:"query=filter;default=equal" validate:"oneof=starts_with ends_with contains equal"`
}

func (c containerHealthyInput) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// Start begins the overseer's operation.
func (o *Overseer) Start() error {
	ctx := context.Background()
	router := chi.NewRouter()

	// Basic middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.NoCache)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(constants.DefaultServerTimeout))
	router.Use(middleware.StripSlashes)
	router.Use(middleware.CleanPath)
	router.Use(middleware.Heartbeat(constants.PingPath))

	// Use common security middleware
	router.Use(security.BasicSecurity)

	router.Get("/favicon.ico", o.handleFavicon)
	router.Get("/assets/favicon.ico", o.handleFavicon) // Add alternative path

	router.Route("/api/v1", func(r chi.Router) {
		// Add token auth middleware for all other routes
		r.Group(func(r chi.Router) {
			r.Use(func(next http.Handler) http.Handler {
				return commonMiddleware.TokenAuth(next, config.Current.Overseer.AuthTokens)
			})

			r.Get("/scouts", o.handleListScouts)
			r.Get("/containers", o.handleContainers)
			r.Get("/images", o.handleImages)
			r.Get("/networks", o.handleNetworks)
			r.Get("/volumes", o.handleVolumes)
			r.With(httpin.NewInput(containerHealthyInput{})).Get("/container/{name}/healthy", o.handleContainerHealth)
		})
	})

	srvAddr := fmt.Sprintf(":%d", constants.DefaultOverseerPort)
	srv := &http.Server{
		Handler:      router,
		Addr:         srvAddr,
		WriteTimeout: config.Current.Server.WriteTimeout,
		ReadTimeout:  config.Current.Server.ReadTimeout,
		IdleTimeout:  config.Current.Server.IdleTimeout,
	}

	slog.InfoContext(ctx, "Overseer started", "address", srvAddr, "provider", o.provider.GetType())

	// Run server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "failed to start server", "error", err)
			errChan <- err
		}
	}()

	// Check for startup errors
	select {
	case err := <-errChan:
		return err
	default:
	}

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(ctx, constants.DefaultServerShutdownGracePeriod)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "Server shutdown failed", "error", err)
		return err
	}

	slog.InfoContext(ctx, "Server shutdown successfully")
	return nil
}

func (o *Overseer) handleFavicon(w http.ResponseWriter, _ *http.Request) {
	// Set correct content type for favicon
	w.Header().Set("Content-Type", "image/x-icon") // Changed from image/png

	// Add additional headers for better browser compatibility
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(assets.Favicon)))

	// Write favicon data
	if _, err := w.Write(assets.Favicon); err != nil {
		slog.ErrorContext(context.Background(), "Failed to serve favicon", "error", err)
	}
}

func (o *Overseer) handleListScouts(w http.ResponseWriter, r *http.Request) {
	scouts, err := o.provider.GetScouts(r.Context())
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, scouts)
}

func (o *Overseer) handleContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := o.provider.GetContainers(r.Context())
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, containers)
}

func (o *Overseer) handleImages(w http.ResponseWriter, r *http.Request) {
	images, err := o.provider.GetImages(r.Context())
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, images)
}

func (o *Overseer) handleNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := o.provider.GetNetworks(r.Context())
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, networks)
}

func (o *Overseer) handleVolumes(w http.ResponseWriter, r *http.Request) {
	volumes, err := o.provider.GetVolumes(r.Context())
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, volumes)
}

func (o *Overseer) handleContainerHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	payload, ok := r.Context().Value(httpin.Input).(*containerHealthyInput)
	if !ok {
		commonHttp.WriteErrorResponse(w, http.StatusBadRequest, ErrInvalidRequestFormat)
		return
	}
	if err := payload.Validate(); err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	containerState, err := o.provider.CheckContainerHealth(ctx, payload.Name, payload.Filter)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	// If no containers were found, return 404
	if len(containerState) == 0 {
		commonHttp.WriteErrorResponse(w, http.StatusNotFound, errors.New("container not found"))
		return
	}

	// Check if all containers are healthy, error if any are not healthy
	for name, state := range containerState {
		if !state {
			commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, fmt.Errorf("container %s is not healthy", name))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
