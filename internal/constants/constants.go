// Package constants defines application-wide constants and configuration values.
package constants

import (
	"time"
)

// Version information for the application.
const (
	// GithubOwner is the owner of the GitHub repository.
	GithubOwner = "hibare"

	// ProgramIdentifier is the identifier for the application.
	ProgramIdentifier = "ArguSwarm"
)

// DefaultScoutPort is the default port number for scout agents.
const DefaultScoutPort = 8081

// DefaultOverseerPort is the default port number for the overseer service.
const DefaultOverseerPort = 8080

// Server timeout related constants.
const (
	// DefaultServerReadTimeout is the maximum duration for reading the entire request.
	DefaultServerReadTimeout = 15 * time.Second

	// DefaultServerWriteTimeout is the maximum duration before timing out writes of the response.
	DefaultServerWriteTimeout = 15 * time.Second

	// DefaultServerIdleTimeout is the maximum amount of time to wait for the next request.
	DefaultServerIdleTimeout = 60 * time.Second

	// DefaultServerShutdownGracePeriod is the duration to wait for server shutdown.
	DefaultServerShutdownGracePeriod = 60 * time.Second

	// DefaultServerTimeout is the overall timeout for server operations.
	DefaultServerTimeout = 60 * time.Second

	// DefaultServerRequestSizeLimit is the maximum size of a request.
	DefaultServerRequestSizeLimit = 1024 * 1024 // 1MB

	// DefaultScoutTask is the default task name for scout agents.
	DefaultScoutTask = "tasks.scout"

	// DefaultConcurrentScoutsQuery is the maximum number of concurrent scout queries.
	DefaultConcurrentScoutsQuery = 10
)

// Health check and retry related constants.
const (
	// DefaultHTTPClientTimeout is the timeout for HTTP client requests.
	DefaultHTTPClientTimeout = 10 * time.Second
)

// API endpoint paths.
const (
	// PingPath is the endpoint for health check pings.
	PingPath = "/ping"
)

// Container state constants.
const (
	// ContainerStateRunning indicates a container is currently running.
	ContainerStateRunning = "running"

	// ContainerStateHealthy indicates a container is in a healthy state.
	ContainerStateHealthy = "healthy"
)
