package config

import (
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/hibare/ArguSwarm/internal/constants"
	commonLogger "github.com/hibare/GoCommon/v2/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected *Config
	}{
		{
			name: "all defaults",
			envVars: map[string]string{
				"ARGUSWARM_SERVER_SHARED_SECRET": "test-secret",
			},
			expected: &Config{
				Scout: ScoutConfig{
					OverseerServerAddress: "",
					Port:                  constants.DefaultScoutPort,
				},
				Overseer: OverseerConfig{
					Port:       constants.DefaultOverseerPort,
					AuthTokens: []string{},
				},
				Logger: LoggerConfig{
					Level: commonLogger.DefaultLoggerLevel,
					Mode:  commonLogger.DefaultLoggerMode,
				},
				Server: ServerConfig{
					ReadTimeout:  constants.DefaultServerReadTimeout,
					WriteTimeout: constants.DefaultServerWriteTimeout,
					IdleTimeout:  constants.DefaultServerIdleTimeout,
					SharedSecret: "test-secret",
				},
				HTTPClient: HTTPClientConfig{
					Timeout: constants.DefaultHTTPClientTimeout,
				},
			},
		},
		{
			name: "custom values",
			envVars: map[string]string{
				"ARGUSWARM_OVERSEER_SERVER_ADDRESS": "custom-address:8080",
				"ARGUSWARM_SCOUT_PORT":              "9090",
				"ARGUSWARM_OVERSEER_PORT":           "8080",
				"ARGUSWARM_LOG_LEVEL":               "debug",
				"ARGUSWARM_LOG_MODE":                "json",
				"ARGUSWARM_SERVER_READ_TIMEOUT":     "10s",
				"ARGUSWARM_SERVER_WRITE_TIMEOUT":    "20s",
				"ARGUSWARM_SERVER_IDLE_TIMEOUT":     "30s",
				"ARGUSWARM_HTTP_CLIENT_TIMEOUT":     "5s",
				"ARGUSWARM_OVERSEER_AUTH_TOKENS":    "token1,token2,token3",
				"ARGUSWARM_SERVER_SHARED_SECRET":    "test-secret",
			},
			expected: &Config{
				Scout: ScoutConfig{
					OverseerServerAddress: "custom-address:8080",
					Port:                  9090,
				},
				Overseer: OverseerConfig{
					Port:       8080,
					AuthTokens: []string{"token1", "token2", "token3"},
				},
				Logger: LoggerConfig{
					Level: "debug",
					Mode:  "json",
				},
				Server: ServerConfig{
					ReadTimeout:  10 * time.Second,
					WriteTimeout: 20 * time.Second,
					IdleTimeout:  30 * time.Second,
					SharedSecret: "test-secret",
				},
				HTTPClient: HTTPClientConfig{
					Timeout: 5 * time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment variables
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			// Load config
			Load()

			// Verify config
			assert.Equal(t, tt.expected, Current)
		})
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	// Set invalid log level
	t.Setenv("ARGUSWARM_LOG_LEVEL", "invalid-level")

	// Test that Load exits with invalid log level
	if os.Getenv("TEST_EXIT") == "1" {
		Load()
		return
	}
	const testName = "TestLoad_InvalidLogLevel"
	// #nosec G204
	cmd := exec.Command(os.Args[0], "-test.run=^"+testName+"$")
	cmd.Env = append(os.Environ(), "TEST_EXIT=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestLoad_InvalidLogMode(t *testing.T) {
	// Set invalid log mode
	t.Setenv("ARGUSWARM_LOG_MODE", "invalid-mode")

	// Test that Load exits with invalid log mode
	if os.Getenv("TEST_EXIT") == "1" {
		Load()
		return
	}
	const testName = "TestLoad_InvalidLogMode"
	// #nosec G204
	cmd := exec.Command(os.Args[0], "-test.run=^"+testName+"$")
	cmd.Env = append(os.Environ(), "TEST_EXIT=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestLoad_MissingSharedSecret(t *testing.T) {
	// Test that Load exits when shared secret is not set
	if os.Getenv("TEST_EXIT") == "1" {
		Load()
		return
	}
	const testName = "TestLoad_MissingSharedSecret"
	// #nosec G204
	cmd := exec.Command(os.Args[0], "-test.run=^"+testName+"$")
	cmd.Env = append(os.Environ(), "TEST_EXIT=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}
