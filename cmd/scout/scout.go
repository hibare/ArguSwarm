package scout

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/constants"
	"github.com/hibare/ArguSwarm/internal/middleware/security"
	"github.com/hibare/ArguSwarm/internal/providers"
	dockerSwarm "github.com/hibare/ArguSwarm/internal/providers/dockerswarm"
	"github.com/hibare/ArguSwarm/internal/providers/types"
	commonMiddleware "github.com/hibare/GoCommon/v2/pkg/http/middleware"
)

// Agent represents a network scout agent.
type Agent struct {
	scoutAgent interface{}
	provider   types.ProviderType
}

// NewAgent creates a new Agent instance for Docker Swarm only.
func NewAgent() (*Agent, error) {
	providerType := providers.DetectProviderType()

	// Scouts are only needed for Docker Swarm
	if providerType != types.ProviderDockerSwarm {
		return nil, fmt.Errorf("scouts are only needed for Docker Swarm, detected provider: %s", providerType)
	}

	scoutAgent, err := dockerSwarm.NewScoutAgent()
	if err != nil {
		return nil, fmt.Errorf("failed to create scout agent: %w", err)
	}

	return &Agent{
		scoutAgent: scoutAgent,
		provider:   providerType,
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
		r.Get("/volumes", s.handleVolumes)
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
	// Only Docker Swarm scouts are supported
	if agent, ok := s.scoutAgent.(*dockerSwarm.ScoutAgent); ok {
		agent.HandleContainers(w, r)
	}
}

func (s *Agent) handleImages(w http.ResponseWriter, r *http.Request) {
	// Only Docker Swarm scouts are supported
	if agent, ok := s.scoutAgent.(*dockerSwarm.ScoutAgent); ok {
		agent.HandleImages(w, r)
	}
}

func (s *Agent) handleNetworks(w http.ResponseWriter, r *http.Request) {
	// Only Docker Swarm scouts are supported
	if agent, ok := s.scoutAgent.(*dockerSwarm.ScoutAgent); ok {
		agent.HandleNetworks(w, r)
	}
}

func (s *Agent) handleVolumes(w http.ResponseWriter, r *http.Request) {
	// Only Docker Swarm scouts are supported
	if agent, ok := s.scoutAgent.(*dockerSwarm.ScoutAgent); ok {
		agent.HandleVolumes(w, r)
	}
}
