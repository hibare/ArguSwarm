<div align="center">
  <img src="./internal/assets/favicon.png" alt="ArguSwarm Logo" width="200" height="200">

# ArguSwarm

_A Distributed Docker Container information System_

[![Go Report Card](https://goreportcard.com/badge/github.com/hibare/ArguSwarm)](https://goreportcard.com/report/github.com/hibare/ArguSwarm)
[![Docker Hub](https://img.shields.io/docker/pulls/hibare/arguswarm)](https://hub.docker.com/r/hibare/arguswarm)
[![Docker image size](https://img.shields.io/docker/image-size/hibare/arguswarm/latest)](https://hub.docker.com/r/hibare/arguswarm)
[![GitHub issues](https://img.shields.io/github/issues/hibare/ArguSwarm)](https://github.com/hibare/ArguSwarm/issues)
[![GitHub pull requests](https://img.shields.io/github/issues-pr/hibare/ArguSwarm)](https://github.com/hibare/ArguSwarm/pulls)
[![GitHub](https://img.shields.io/github/license/hibare/ArguSwarm)](https://github.com/hibare/ArguSwarm/blob/main/LICENSE)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/hibare/ArguSwarm)](https://github.com/hibare/ArguSwarm/releases)

</div>

ArguSwarm is a distributed container monitoring system designed for both Docker Swarm and Kubernetes environments. It consists of an overseer service that runs on manager nodes and scout agents that run on worker nodes to collect and aggregate container information across the cluster.

## Features

- **Multi-Platform Support**: Works with both Docker Swarm and Kubernetes
- **Distributed Monitoring**: Container monitoring across cluster nodes
- **REST API**: Query container state and health via HTTP API
- **Health Monitoring**: Container health checks with customizable filters
- **Real-time Data**: Live container, image, network, and volume information
- **Secure Communication**: Shared secret authentication between components
- **Horizontal Scalability**: Scout agents run on each node
- **Parallel Processing**: Configurable concurrent query execution
- **Auto-Detection**: Automatically detects the orchestration platform

## Architecture

- **Overseer**: Central service that runs on manager nodes

  - Manages scout discovery and health monitoring
  - Aggregates container information from all scouts
  - Provides REST API endpoints for clients
  - Handles authentication and security
  - Supports both Docker Swarm and Kubernetes

- **Scout**: Agent service that runs on worker nodes (Docker Swarm only)
  - Collects local container information
  - Monitors container health status
  - Exposes REST API for overseer queries
  - Reports node-specific metrics
  - Not needed for Kubernetes deployments

## Supported Platforms

### Docker Swarm

- Uses Docker API to query containers, images, networks, and volumes
- Discovers scouts via DNS lookup (`tasks.scout`)
- Deployed using Docker Swarm stack

### Kubernetes

- Uses Kubernetes API to query pods, services, and persistent volumes
- Discovers nodes via Kubernetes API
- Deployed using Kubernetes manifests or Helm charts

## Installation

### Docker Swarm Deployment

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
   ARGUSWARM_PROVIDER=docker-swarm  # Optional: auto-detected if not set
   ```

4. Deploy to Docker Swarm:

   ```bash
   docker stack deploy -c docker-compose.yml arguswarm
   ```

### Kubernetes Deployment

1. Clone the repository:

   ```bash
   git clone https://github.com/hibare/ArguSwarm.git
   cd ArguSwarm
   ```

2. Create the namespace and RBAC:

   ```bash
   kubectl apply -f k8s/namespace.yaml
   kubectl apply -f k8s/rbac.yaml
   ```

3. Create secrets (replace with your values):

   ```bash
   kubectl create secret generic arguswarm-secret \
     --from-literal=ARGUSWARM_SERVER_SHARED_SECRET=<your-secret> \
     --from-literal=ARGUSWARM_OVERSEER_AUTH_TOKENS=<token1,token2> \
     -n arguswarm
   ```

4. Deploy the application:

   ```bash
   kubectl apply -f k8s/configmap.yaml
   kubectl apply -f k8s/overseer-deployment.yaml
   # Note: Scout daemonset not needed for Kubernetes
   ```

5. Access the service:

   ```bash
   kubectl port-forward -n arguswarm svc/arguswarm-overseer 8080:8080
   ```

### Configuration

ArguSwarm can be configured using environment variables:

| Variable                              | Description                                           | Default |
| ------------------------------------- | ----------------------------------------------------- | ------- |
| ARGUSWARM_PROVIDER                    | Orchestration platform (docker-swarm/kubernetes/auto) | auto    |
| ARGUSWARM_OVERSEER_PORT               | Overseer service port                                 | 8080    |
| ARGUSWARM_SCOUT_PORT                  | Scout service port                                    | 8081    |
| ARGUSWARM_MAX_CONCURRENT_SCOUTS_QUERY | Max parallel scout queries                            | 10      |
| ARGUSWARM_LOG_LEVEL                   | Logging level                                         | info    |
| ARGUSWARM_LOG_MODE                    | Logging mode (text/json)                              | text    |

## API Documentation

Overseer API Endpoints

- `GET /api/v1/scouts` - List all connected scouts (Docker Swarm only)
- `GET /api/v1/containers` - List all containers (containers/pods) across the cluster
- `GET /api/v1/images` - List all images across the cluster
- `GET /api/v1/networks` - List all networks across the cluster
- `GET /api/v1/volumes` - List all volumes across the cluster
- `GET /api/v1/container/{name}/healthy` - Check container health status

All API endpoints require authentication using the configured auth tokens.

Scout API Endpoints (Docker Swarm only)

- `GET /api/v1/containers` - List containers on the node
- `GET /api/v1/images` - List images on the node
- `GET /api/v1/networks` - List networks on the node
- `GET /api/v1/volumes` - List volumes on the node

Note: Scouts are not needed for Kubernetes deployments as the overseer queries the Kubernetes API directly.

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
