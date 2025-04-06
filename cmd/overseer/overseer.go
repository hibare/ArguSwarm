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
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/constants"
	"github.com/hibare/ArguSwarm/internal/utils"
	commonHttp "github.com/hibare/GoCommon/v2/pkg/http"
)

// Overseer manages the health and status of scout agents.
type Overseer struct {
	scoutStore *ScoutStore
	context    context.Context
	httpClient *http.Client
}

// HealthStatus represents the health status of a scout.
type HealthStatus struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	NodeID    string    `json:"node_id"`
}

// NewOverseer creates a new Overseer instance.
func NewOverseer() (*Overseer, error) {
	return &Overseer{
		scoutStore: NewScoutStore(),
		context:    context.Background(),
		httpClient: &http.Client{Timeout: config.Current.HTTPClient.Timeout},
	}, nil
}

// Start begins the overseer's operation.
func (o *Overseer) Start() error {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.RequestID)
	router.Use(middleware.NoCache)
	router.Use(middleware.RealIP)
	router.Use(middleware.Timeout(constants.DefaultServerTimeout))
	router.Use(middleware.StripSlashes)
	router.Use(middleware.CleanPath)
	router.Use(middleware.Heartbeat(constants.PingPath))

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/scouts/ping", o.handleScoutPing)
		r.Get("/scouts", o.handleListScouts)
		r.Get("/scouts/active", o.handleActiveScouts)
		r.Get("/containers", o.handleContainers)
		r.Get("/images", o.handleImages)
		r.Get("/networks", o.handleNetworks)
		r.Get("/volumes", o.handleVolumes)
		r.Get("/container/{name}/healthy", o.handleContainerHealth)
	})

	srvAddr := fmt.Sprintf(":%d", constants.DefaultOverseerPort)
	srv := &http.Server{
		Handler:      router,
		Addr:         srvAddr,
		WriteTimeout: config.Current.Server.WriteTimeout,
		ReadTimeout:  config.Current.Server.ReadTimeout,
		IdleTimeout:  config.Current.Server.IdleTimeout,
	}

	slog.InfoContext(o.context, "Overseer started", "address", srvAddr)

	// Start health check routine
	go o.scoutStore.CheckHealth(constants.DefaultHealthCheckInterval)

	// Run server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(o.context, "failed to start server", "error", err)
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

	ctx, cancel := context.WithTimeout(o.context, constants.DefaultServerShutdownGracePeriod)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.ErrorContext(o.context, "Server shutdown failed", "error", err)
		return err
	}

	slog.InfoContext(o.context, "Server shutdown successfully")
	return nil
}

func (o *Overseer) fetchResource(s *Scout, resourceType string) ([]any, error) {
	url := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(s.Address, strconv.Itoa(config.Current.Scout.Port)),
		Path:   "/api/v1/" + resourceType,
	}

	req, err := http.NewRequestWithContext(o.context, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.ErrorContext(o.context, "Failed to close response body", "error", closeErr)
		}
	}()

	var result []any
	if decodeErr := json.NewDecoder(resp.Body).Decode(&result); decodeErr != nil {
		return nil, decodeErr
	}

	return result, nil
}

func (o *Overseer) handleScoutPing(w http.ResponseWriter, r *http.Request) {
	var status HealthStatus
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		commonHttp.WriteErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		slog.ErrorContext(o.context, "Failed to parse remote address", "error", err)
		commonHttp.WriteErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	o.scoutStore.UpdateScout(status.NodeID, host)
	w.WriteHeader(http.StatusOK)
}

func (o *Overseer) handleListScouts(w http.ResponseWriter, _ *http.Request) {
	scouts := o.scoutStore.GetAllScouts()
	commonHttp.WriteJsonResponse(w, http.StatusOK, scouts)
}

func (o *Overseer) handleActiveScouts(w http.ResponseWriter, _ *http.Request) {
	scouts := o.scoutStore.GetActiveScouts()
	commonHttp.WriteJsonResponse(w, http.StatusOK, scouts)
}

func (o *Overseer) handleContainers(w http.ResponseWriter, _ *http.Request) {
	scouts := o.scoutStore.GetActiveScouts()
	if len(scouts) == 0 {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, errors.New("no active scouts available"))
		return
	}

	worker := func(s *Scout) ([]any, error) {
		return o.fetchResource(s, "containers")
	}

	results, errors := utils.ParallelExecute(scouts, worker)

	for _, err := range errors {
		slog.ErrorContext(o.context, "Error fetching containers", "error", err)
	}

	allContainers := make([]any, 0)
	for _, containers := range results {
		allContainers = append(allContainers, containers...)
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, allContainers)
}

func (o *Overseer) handleImages(w http.ResponseWriter, _ *http.Request) {
	scouts := o.scoutStore.GetActiveScouts()
	if len(scouts) == 0 {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, errors.New("no active scouts available"))
		return
	}

	worker := func(s *Scout) ([]any, error) {
		return o.fetchResource(s, "images")
	}

	results, errors := utils.ParallelExecute(scouts, worker)

	for _, err := range errors {
		slog.ErrorContext(o.context, "Error fetching images", "error", err)
	}

	allImages := make([]any, 0)
	for _, images := range results {
		allImages = append(allImages, images...)
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, allImages)
}

func (o *Overseer) handleNetworks(w http.ResponseWriter, _ *http.Request) {
	scouts := o.scoutStore.GetActiveScouts()
	if len(scouts) == 0 {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, errors.New("no active scouts available"))
		return
	}

	worker := func(s *Scout) ([]any, error) {
		return o.fetchResource(s, "networks")
	}

	results, errors := utils.ParallelExecute(scouts, worker)

	for _, err := range errors {
		slog.ErrorContext(o.context, "Error fetching networks", "error", err)
	}

	allNetworks := make([]any, 0)
	for _, networks := range results {
		allNetworks = append(allNetworks, networks...)
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, allNetworks)
}

func (o *Overseer) handleVolumes(w http.ResponseWriter, _ *http.Request) {
	scouts := o.scoutStore.GetActiveScouts()
	if len(scouts) == 0 {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, errors.New("no active scouts available"))
		return
	}

	worker := func(s *Scout) ([]any, error) {
		return o.fetchResource(s, "volumes")
	}

	results, errors := utils.ParallelExecute(scouts, worker)

	for _, err := range errors {
		slog.ErrorContext(o.context, "Error fetching volumes", "error", err)
	}

	allVolumes := make([]any, 0)
	for _, volumes := range results {
		allVolumes = append(allVolumes, volumes...)
	}

	commonHttp.WriteJsonResponse(w, http.StatusOK, allVolumes)
}

func (o *Overseer) handleContainerHealth(w http.ResponseWriter, r *http.Request) {
	containerState := map[string]bool{}

	containerName := chi.URLParam(r, "name")
	if containerName == "" {
		commonHttp.WriteErrorResponse(w, http.StatusBadRequest, errors.New("container name is required"))
		return
	}

	scouts := o.scoutStore.GetActiveScouts()
	if len(scouts) == 0 {
		commonHttp.WriteErrorResponse(w, http.StatusServiceUnavailable, errors.New("no active scouts available"))
		return
	}

	worker := func(s *Scout) (bool, error) {
		containersRaw, err := o.fetchResource(s, "containers")
		if err != nil {
			return false, err
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
		slog.ErrorContext(o.context, "Error checking container health", "error", err)
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
