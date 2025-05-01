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
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/ggicci/httpin"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/hibare/ArguSwarm/internal/assets"
	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/constants"
	"github.com/hibare/ArguSwarm/internal/middleware/security"
	commonConcurrency "github.com/hibare/GoCommon/v2/pkg/concurrency"
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

func (o *Overseer) fetchResource(ctx context.Context, scoutIP string, resourceType string) ([]any, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(scoutIP, strconv.Itoa(config.Current.Scout.Port)),
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

func (o *Overseer) queryScouts(ctx context.Context, resource string) ([]any, error) {
	scouts, err := o.getScouts(ctx)
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
		scoutIP := s // capture range variable
		tasks[i] = commonConcurrency.ParallelTask{
			Name: fmt.Sprintf("Scout-%s", scoutIP),
			Task: func(ctx context.Context) error {
				result, rErr := o.fetchResource(ctx, scoutIP, resource)
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

func (o *Overseer) handleListScouts(w http.ResponseWriter, r *http.Request) {
	scouts, err := o.getScouts(r.Context())
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, scouts)
}

func (o *Overseer) handleContainers(w http.ResponseWriter, r *http.Request) {
	results, err := o.queryScouts(r.Context(), "containers")
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, results)
}

func (o *Overseer) handleImages(w http.ResponseWriter, r *http.Request) {
	results, err := o.queryScouts(r.Context(), "images")
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, results)
}

func (o *Overseer) handleNetworks(w http.ResponseWriter, r *http.Request) {
	results, err := o.queryScouts(r.Context(), "networks")
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, results)
}

func (o *Overseer) handleVolumes(w http.ResponseWriter, r *http.Request) {
	results, err := o.queryScouts(r.Context(), "volumes")
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, results)
}

func (o *Overseer) handleContainerHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	containerState := map[string]bool{}

	payload, ok := r.Context().Value(httpin.Input).(*containerHealthyInput)
	if !ok {
		commonHttp.WriteErrorResponse(w, http.StatusBadRequest, ErrInvalidRequestFormat)
		return
	}
	if err := payload.Validate(); err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	results, err := o.queryScouts(ctx, "containers")
	if err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, err)
		return
	}

	containers := make([]container.Summary, len(results))
	for i, raw := range results {
		// Convert map to JSON and then to container.Summary struct
		jsonData, jsonErr := json.Marshal(raw)
		if jsonErr != nil {
			commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, jsonErr)
			return
		}
		if jsonErr1 := json.Unmarshal(jsonData, &containers[i]); jsonErr1 != nil {
			commonHttp.WriteErrorResponse(w, http.StatusInternalServerError, jsonErr1)
			return
		}
	}

	// Check if the container name matches the filter
	// and update the state accordingly
	for _, container := range containers {
		for _, name := range container.Names {
			name = strings.TrimPrefix(name, "/")

			matches := map[string]func(string, string) bool{
				"starts_with": strings.HasPrefix,
				"ends_with":   strings.HasSuffix,
				"contains":    strings.Contains,
				"equal":       func(s1, s2 string) bool { return s1 == s2 },
			}

			if matchFunc, ok := matches[payload.Filter]; ok && matchFunc(name, payload.Name) { //nolint:govet // This is acceptable
				state := container.State == constants.ContainerStateRunning ||
					container.State == constants.ContainerStateHealthy

				containerState[name] = state
			}
		}
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
