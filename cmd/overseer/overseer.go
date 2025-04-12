package overseer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/constants"
	"github.com/hibare/ArguSwarm/internal/middleware/security"
	"github.com/hibare/ArguSwarm/internal/utils"
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

// Overseer manages the health and status of scout agents.
type Overseer struct {
	httpClient *http.Client
}

// NewOverseer creates a new Overseer instance.
func NewOverseer() (*Overseer, error) {
	return &Overseer{
		httpClient: &http.Client{Timeout: config.Current.HTTPClient.Timeout},
	}, nil
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
			r.Get("/container/{name}/healthy", o.handleContainerHealth)
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

	slog.InfoContext(ctx, "Overseer started", "address", srvAddr)

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

func (o *Overseer) getScouts(ctx context.Context) ([]string, error) {
	ips, err := net.LookupIP("tasks.scout")
	if err != nil {
		return nil, err
	}

	scouts := make([]string, len(ips))
	for i, ip := range ips {
		scouts[i] = ip.String()
	}

	slog.DebugContext(ctx, "Scouts", "scouts", scouts, "count", len(scouts))

	return scouts, nil
}

func (o *Overseer) fetchResource(ctx context.Context, ip string, resourceType string) ([]any, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(ip, strconv.Itoa(config.Current.Scout.Port)),
		Path:   "/api/v1/" + resourceType,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create request", "error", err)
		return nil, ErrFailedToCreateRequest
	}

	req.Header.Set(commonMiddleware.AuthHeaderName, config.Current.Server.SharedSecret)
	req.Header.Set("User-Agent", "ArguSwarm-Overseer/1.0")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to fetch resource", "error", err)
		return nil, ErrFailedToFetchResource
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "Failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		slog.ErrorContext(ctx, "Unexpected status code", "status", resp.StatusCode)
		return nil, ErrUnexpectedStatusCode
	}

	var result []any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		slog.ErrorContext(ctx, "Failed to decode response", "error", decodeErr)
		return nil, ErrFailedToDecodeResponse
	}

	return result, nil
}

func (o *Overseer) handleListScouts(w http.ResponseWriter, r *http.Request) {
	scouts, err := o.getScouts(r.Context())
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, scouts)
}

func (o *Overseer) handleContainers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	scouts, err := o.getScouts(ctx)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	worker := func(s string) ([]any, error) {
		return o.fetchResource(ctx, s, "containers")
	}

	results, errors := utils.ParallelExecute(scouts, worker)

	for _, err := range errors {
		slog.ErrorContext(ctx, "Error fetching containers", "error", err)
	}

	allContainers := make([]any, 0)
	for _, containers := range results {
		allContainers = append(allContainers, containers...)
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, allContainers)
}

func (o *Overseer) handleImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	scouts, err := o.getScouts(ctx)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	worker := func(s string) ([]any, error) {
		return o.fetchResource(ctx, s, "images")
	}

	results, errors := utils.ParallelExecute(scouts, worker)

	for _, err := range errors {
		slog.ErrorContext(ctx, "Error fetching images", "error", err)
	}

	allImages := make([]any, 0)
	for _, images := range results {
		allImages = append(allImages, images...)
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, allImages)
}

func (o *Overseer) handleNetworks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	scouts, err := o.getScouts(ctx)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	worker := func(s string) ([]any, error) {
		return o.fetchResource(ctx, s, "networks")
	}

	results, errors := utils.ParallelExecute(scouts, worker)

	for _, err := range errors {
		slog.ErrorContext(ctx, "Error fetching networks", "error", err)
	}

	allNetworks := make([]any, 0)
	for _, networks := range results {
		allNetworks = append(allNetworks, networks...)
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, allNetworks)
}

func (o *Overseer) handleVolumes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	scouts, err := o.getScouts(ctx)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	worker := func(s string) ([]any, error) {
		return o.fetchResource(ctx, s, "volumes")
	}

	results, errors := utils.ParallelExecute(scouts, worker)

	for _, err := range errors {
		slog.ErrorContext(ctx, "Error fetching volumes", "error", err)
	}

	allVolumes := make([]any, 0)
	for _, volumes := range results {
		allVolumes = append(allVolumes, volumes...)
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, allVolumes)
}

func (o *Overseer) handleContainerHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	containerState := map[string]bool{}

	containerName := chi.URLParam(r, "name")
	if containerName == "" {
		commonHttp.WriteErrorResponse(w, http.StatusBadRequest, errors.New("container name is required"))
		return
	}

	scouts, err := o.getScouts(ctx)
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	worker := func(s string) (bool, error) {
		containersRaw, cErr := o.fetchResource(ctx, s, "containers")
		if cErr != nil {
			return false, cErr
		}

		// Convert []any to []container.Summary
		containers := make([]container.Summary, len(containersRaw))
		for i, raw := range containersRaw {
			// Convert map to JSON and then to container.Summary struct
			jsonData, jsonErr := json.Marshal(raw)
			if jsonErr != nil {
				return false, jsonErr
			}
			if jsonErr1 := json.Unmarshal(jsonData, &containers[i]); jsonErr1 != nil {
				return false, jsonErr1
			}
		}

		for _, container := range containers {
			for _, name := range container.Names {
				if strings.TrimPrefix(name, "/") == containerName {
					containerState[name] = container.State == constants.ContainerStateRunning ||
						container.State == constants.ContainerStateHealthy
				}
			}
		}
		return false, nil
	}

	_, errs := utils.ParallelExecute(scouts, worker)

	for _, err := range errs {
		slog.ErrorContext(ctx, "Error checking container health", "error", err)
	}

	// check containerState map, if all values are true, return 200 else 503
	if len(containerState) == 0 {
		commonHttp.WriteErrorResponse(w, http.StatusNotFound, errors.New("container not found"))
		return
	}

	for _, isHealthy := range containerState {
		if !isHealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
