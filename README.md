<div align="center">
  <img src="./internal/assets/favicon.png" alt="ArguSwarm Logo" width="200" height="200">

  # ArguSwarm

  *A Distributed Docker Container information System*

[![Go Report Card](https://goreportcard.com/badge/github.com/hibare/ArguSwarm)](https://goreportcard.com/report/github.com/hibare/ArguSwarm)
[![Docker Hub](https://img.shields.io/docker/pulls/hibare/arguswarm)](https://hub.docker.com/r/hibare/arguswarm)
[![Docker image size](https://img.shields.io/docker/image-size/hibare/arguswarm/latest)](https://hub.docker.com/r/hibare/arguswarm)
[![GitHub issues](https://img.shields.io/github/issues/hibare/ArguSwarm)](https://github.com/hibare/ArguSwarm/issues)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/hibare/ArguSwarm)](https://github.com/hibare/ArguSwarm/pulls)
[![GitHub](https://img.shields.io/github/license/hibare/ArguSwarm)](https://github.com/hibare/ArguSwarm/blob/main/LICENSE)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/hibare/ArguSwarm)](https://github.com/hibare/ArguSwarm/releases)

</div>

ArguSwarm is a distributed Docker container info system designed for Docker Swarm environments. It consists of an overseer service that runs on manager nodes and scout agents that run on worker nodes to collect and aggregate container information across the swarm.

## Features

- Distributed container monitoring across Docker Swarm cluster
- REST API for querying container state and health
- Container health monitoring with customizable filters
- Real-time container, image, network, and volume information
- Secure communication between components using shared secrets
- Horizontal scalability with scout agents running on each node
- Parallel query execution with configurable concurrency

## Architecture

- **Overseer**: Central service that runs on manager nodes

  - Manages scout discovery and health monitoring
  - Aggregates container information from all scouts
  - Provides REST API endpoints for clients
  - Handles authentication and security

- **Scout**: Agent service that runs on worker nodes
  - Collects local container information
  - Monitors container health status
  - Exposes REST API for overseer queries
  - Reports node-specific metrics

## Installation

### Using Docker Compose

1. Clone the repository:

   ```bash
   git clone https://github.com/hibare/ArguSwarm.git
   cd ArguSwarm
   ```

2. Copy and configure environment variables:

   ```bash
   cp .env.example .env
   ```

3. Configure the following required environment variables:

   ```bash
   ARGUSWARM_SERVER_SHARED_SECRET=<your-secret>  # Shared secret for overseer-scout communication
   ARGUSWARM_OVERSEER_AUTH_TOKENS=<token1,token2> # Auth tokens for API access
   ```

4. Deploy to Docker Swarm:

   ```bash
   docker stack deploy -c docker-compose.yml arguswarm
   ```

### Configuration

ArguSwarm can be configured using environment variables:

| Variable                              | Description                | Default |
| ------------------------------------- | -------------------------- | ------- |
| ARGUSWARM_OVERSEER_PORT               | Overseer service port      | 8080    |
| ARGUSWARM_SCOUT_PORT                  | Scout service port         | 8081    |
| ARGUSWARM_MAX_CONCURRENT_SCOUTS_QUERY | Max parallel scout queries | 10      |
| ARGUSWARM_LOG_LEVEL                   | Logging level              | info    |
| ARGUSWARM_LOG_MODE                    | Logging mode (text/json)   | text    |

## API Documentation

Overseer API Endpoints

- `GET /api/v1/scouts` - List all connected scouts
- `GET /api/v1/containers` - List all containers across the swarm
- `GET /api/v1/images` - List all images across the swarm
- `GET /api/v1/networks` - List all networks across the swarm
- `GET /api/v1/volumes` - List all volumes across the swarm
- `GET /api/v1/container/{name}/healthy` - Check container health status

All API endpoints require authentication using the configured auth tokens.

Scout API Endpoints

- `GET /api/v1/containers` - List containers on the node
- `GET /api/v1/images` - List images on the node
- `GET /api/v1/networks`- List networks on the node
- `GET /api/v1/volumes` - List volumes on the node

## Development

### Prerequisites

- Go 1.24 or later
- Docker
- Docker Compose
- Make

### Setup Development Environment

1. Initialize the project:

   ```bash
   make init
   ```

2. Start development servers:

   ```bash
   make dev
   ```

3. Run tests:

   ```bash
   make test
   ```

### Project Structure

- `cmd` - Main application entry points
- `internal` - Internal packages
- `openapi` - API specifications
- `.air` - Development server configurations
- `docker-compose.dev.yml` - Development environment setup

## Contributing

- Fork the repository
- Create your feature branch
- Commit your changes
- Push to the branch
- Create a new Pull Request
