# Deployment Guide

This guide shows how to deploy the simple microservices application for testing Beyla distributed tracing.

## Prerequisites

- Docker
- Kubernetes cluster (Minikube, K3s, or similar)
- kubectl configured
- Go 1.24+ (for local development)

## Local Development

### 1. Build and Run Locally

```bash
# Build all services
make build

# Run all services locally
make run-local

# Test the service chain
make test

# Stop services
make stop-local
```

### 2. Test the Service Chain

```bash
# Create a user (triggers full trace chain)
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com"}'

# Get user by ID
curl http://localhost:8080/users/1

# List all users
curl http://localhost:8080/users
```

## Kubernetes Deployment

### 1. Build Docker Images

For Minikube:
```bash
# Use minikube's Docker daemon
eval $(minikube docker-env)

# Build images
make docker-build
```

For K3s or other clusters:
```bash
# Build and push to your registry
make docker-build
docker tag simple-microservices/gateway your-registry/gateway:latest
docker tag simple-microservices/user-service your-registry/user-service:latest
docker tag simple-microservices/data-service your-registry/data-service:latest
docker push your-registry/gateway:latest
docker push your-registry/user-service:latest
docker push your-registry/data-service:latest

# Update k8s/*.yaml files to use your registry
```

### 2. Deploy to Kubernetes

```bash
# Deploy all services
make deploy

# Check deployment status
kubectl get pods -n simple-microservices
kubectl get services -n simple-microservices
```

### 3. Test Kubernetes Deployment

```bash
# For Minikube
minikube service gateway -n simple-microservices

# Or use port-forwarding
kubectl port-forward service/gateway 8080:8080 -n simple-microservices

# Test with curl
make test-k8s
```

## Testing with Beyla

### 1. Deploy Servicegraph Helm Chart

```bash
# From the servicegraph-helm directory
helm install servicegraph ../servicegraph-helm \
  --set global.namespace=simple-microservices \
  --set beyla.discovery.namespace=simple-microservices \
  --set otelCollector.export.endpoint=http://your-tracing-backend:4317
```

### 2. Generate Load for Tracing

```bash
# Generate test load
./scripts/test-load.sh http://localhost:8080 20

# For Kubernetes (after port-forwarding)
./scripts/test-load.sh http://localhost:8080 20
```

### 3. Verify Traces

Check your tracing backend (Jaeger, Tempo, etc.) for traces showing:

1. **Gateway Service** - Initial request handling
2. **User Service** - User creation and data processing coordination  
3. **Data Service** - Data processing with multiple steps

Each trace should show the complete request flow across all 3 services.

## Service Endpoints

- **Gateway**: `:8080`
  - `POST /users` - Create user (triggers full chain)
  - `GET /users/{id}` - Get user by ID
  - `GET /users` - List all users
  - `GET /health` - Health check

- **User Service**: `:8081`
  - `POST /users` - Create user
  - `GET /users/{id}` - Get user by ID
  - `GET /users` - List all users
  - `GET /health` - Health check

- **Data Service**: `:8082`
  - `POST /process` - Process user data
  - `GET /status` - Service status
  - `GET /health` - Health check

## Troubleshooting

### Check Service Logs

```bash
# Local
tail -f logs/*.log

# Kubernetes
kubectl logs -f deployment/gateway -n simple-microservices
kubectl logs -f deployment/user-service -n simple-microservices
kubectl logs -f deployment/data-service -n simple-microservices
```

### Verify Service Communication

```bash
# Test individual services
curl http://localhost:8082/health  # Data Service
curl http://localhost:8081/health  # User Service
curl http://localhost:8080/health  # Gateway

# In Kubernetes
kubectl exec -it deployment/gateway -n simple-microservices -- wget -qO- http://user-service:8081/health
```

### Clean Up

```bash
# Local
make stop-local
make clean

# Kubernetes
make undeploy
```
