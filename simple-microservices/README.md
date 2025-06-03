# Simple Microservices for Beyla Tracing Demo

A minimal 3-service application designed to test distributed tracing with Beyla and OpenTelemetry.

## Architecture

```
┌─────────────┐    HTTP     ┌─────────────┐    HTTP     ┌─────────────┐
│             │   /users    │             │  /process   │             │
│ API Gateway ├────────────►│ User Service├────────────►│Data Service │
│   (8080)    │             │   (8081)    │             │   (8082)    │
└─────────────┘             └─────────────┘             └─────────────┘
```

## Services

1. **API Gateway** (Port 8080) - Entry point, routes requests and aggregates responses
2. **User Service** (Port 8081) - Manages user data and calls Data Service for processing
3. **Data Service** (Port 8082) - Processes data and simulates database operations

## Service Flow

When you call `POST /users` on the API Gateway:
1. Gateway validates request and calls User Service
2. User Service creates user and calls Data Service to process user data
3. Data Service simulates data processing with artificial delays
4. Response flows back through the chain

This creates a clear trace spanning all 3 services.

## Quick Start

### Automated Deployment (Recommended)

```bash
# Deploy everything to Minikube with one command
./deploy-tracing-demo.sh deploy

# Run tests to generate traces
./scripts/quick-test.sh test

# Generate continuous load for testing
./scripts/quick-test.sh load 120

# View deployment status
./deploy-tracing-demo.sh status

# Clean up everything
./deploy-tracing-demo.sh cleanup
```

### Manual Deployment

```bash
# Build all services
make build

# Run locally for testing
make run-local

# Test the service chain
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com"}'

# Deploy to Kubernetes
make deploy

# Clean up
make clean
```

## Testing Traces

1. Deploy the servicegraph Helm chart
2. Deploy these microservices
3. Make API calls to generate traces
4. Observe traces in your tracing backend

The services include artificial delays and multiple HTTP calls to create interesting trace spans.
