// Package config provides configuration management for the application.
package config

import (
	"log"
	"time"

	"github.com/hibare/ArguSwarm/internal/constants"
	"github.com/hibare/GoCommon/v2/pkg/env"
	commonLogger "github.com/hibare/GoCommon/v2/pkg/logger"
)

// LoggerConfig defines logging configuration parameters.
type LoggerConfig struct {
	Level string
	Mode  string
}

// ServerConfig defines server configuration parameters.
type ServerConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// HTTPClientConfig defines HTTP client configuration parameters.
type HTTPClientConfig struct {
	Timeout time.Duration
}

// OverseerConfig defines overseer configuration parameters.
type OverseerConfig struct {
	Port int
}

// ScoutConfig defines scout configuration parameters.
type ScoutConfig struct {
	OverseerServerAddress string
	Port                  int
}

// Config represents the complete application configuration.
type Config struct {
	Logger     LoggerConfig
	Server     ServerConfig
	HTTPClient HTTPClientConfig
	Overseer   OverseerConfig
	Scout      ScoutConfig
}

// Current holds the active application configuration.
var Current *Config

// Load initializes and loads the application configuration.
func Load() {
	env.Load()

	Current = &Config{
		Scout: ScoutConfig{
			OverseerServerAddress: env.MustString("ARGUSWARM_OVERSEER_SERVER_ADDRESS", ""),
			Port:                  env.MustInt("ARGUSWARM_SCOUT_PORT", constants.DefaultScoutPort),
		},
		Overseer: OverseerConfig{
			Port: env.MustInt("ARGUSWARM_OVERSEER_PORT", constants.DefaultOverseerPort),
		},
		Logger: LoggerConfig{
			Level: env.MustString("ARGUSWARM_LOG_LEVEL", commonLogger.DefaultLoggerLevel),
			Mode:  env.MustString("ARGUSWARM_LOG_MODE", commonLogger.DefaultLoggerMode),
		},
		Server: ServerConfig{
			ReadTimeout:  env.MustDuration("ARGUSWARM_SERVER_READ_TIMEOUT", constants.DefaultServerReadTimeout),
			WriteTimeout: env.MustDuration("ARGUSWARM_SERVER_WRITE_TIMEOUT", constants.DefaultServerWriteTimeout),
			IdleTimeout:  env.MustDuration("ARGUSWARM_SERVER_IDLE_TIMEOUT", constants.DefaultServerIdleTimeout),
		},
		HTTPClient: HTTPClientConfig{
			Timeout: env.MustDuration("ARGUSWARM_HTTP_CLIENT_TIMEOUT", constants.DefaultHTTPClientTimeout),
		},
	}

	if !commonLogger.IsValidLogLevel(Current.Logger.Level) {
		log.Fatal("Error invalid logger level")
	}

	if !commonLogger.IsValidLogMode(Current.Logger.Mode) {
		log.Fatal("Error invalid logger mode")
	}

	commonLogger.InitLogger(&Current.Logger.Level, &Current.Logger.Mode)
}
