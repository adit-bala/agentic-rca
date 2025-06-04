# Distributed Tracing Stack Deployment Guide

This guide explains how to deploy and operate the complete distributed tracing system with Grafana Beyla, OpenTelemetry Collector, and ServiceGraph Builder.

## 🏗️ Architecture Overview

```
Microservices → Beyla (DaemonSet) → OTel Collector → ServiceGraph Builder
     ↓              ↓                    ↓                    ↓
HTTP requests   eBPF tracing        OTLP gRPC          "Converted SimpleSpan"
                                   (port 4317/4318)        logs (port 8083)
```

## 🚀 Quick Start

### Option 1: One-Command Deployment
```bash
./quick-start.sh deploy
```

### Option 2: Step-by-Step
```bash
# 1. Deploy everything
./deploy-tracing-stack.sh

# 2. Generate test traffic
./quick-start.sh test

# 3. Check logs
./quick-start.sh logs
```

## 📋 Prerequisites

- **Docker**: For building container images
- **Minikube**: Local Kubernetes cluster
- **kubectl**: Kubernetes CLI
- **Helm**: Package manager for Kubernetes

## 🔧 Manual Commands Explained

### 1. Minikube Setup
```bash
# Start minikube cluster
minikube start --driver=docker

# Check status
minikube status
```

### 2. Build Docker Images
```bash
# Build servicegraph-builder
cd servicegraph-builder
./build.sh docker

# Build microservices
cd simple-microservices
make docker-build

# Load images into minikube
minikube image load servicegraph-builder:v2
minikube image load simple-microservices/gateway:latest
minikube image load simple-microservices/user-service:latest
minikube image load simple-microservices/data-service:latest
```

### 3. Deploy Tracing Stack
```bash
# Deploy Helm chart (Beyla + OTel Collector + ServiceGraph Builder)
helm install tracing-demo ./servicegraph-helm \
  --namespace tracing \
  --create-namespace \
  --wait

# Check deployment
kubectl get pods -n tracing
```

### 4. Deploy Microservices
```bash
# Deploy sample microservices
kubectl apply -f simple-microservices/k8s/ --namespace simple-microservices

# Wait for readiness
kubectl wait --for=condition=ready pod -l app=gateway --namespace simple-microservices --timeout=120s
```

### 5. Test the System
```bash
# Set up port forwarding
kubectl port-forward service/gateway 8080:8080 --namespace simple-microservices &

# Generate test traffic
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Test User","email":"test@example.com"}'

# Check traces
kubectl logs -l app.kubernetes.io/component=servicegraph-builder -n tracing | grep "Converted SimpleSpan"
```

## 🛠️ Available Scripts

### `deploy-tracing-stack.sh`
Complete deployment automation with options:
```bash
./deploy-tracing-stack.sh                    # Full deployment
./deploy-tracing-stack.sh --skip-build       # Skip Docker builds
./deploy-tracing-stack.sh --skip-test        # Skip testing
./deploy-tracing-stack.sh --cleanup          # Clean up after completion
```

### `quick-start.sh`
Quick operations for common tasks:
```bash
./quick-start.sh deploy        # Full deployment
./quick-start.sh build         # Build images only
./quick-start.sh test          # Generate test traffic
./quick-start.sh logs          # View component logs
./quick-start.sh status        # Show pod status
./quick-start.sh cleanup       # Remove everything
./quick-start.sh port-forward  # Set up port forwarding
```

## 🔍 Verification Commands

### Check Component Status
```bash
# All tracing components
kubectl get pods -n tracing

# Microservices
kubectl get pods -n simple-microservices

# Services
kubectl get svc -n tracing
kubectl get svc -n simple-microservices
```

### View Logs
```bash
# ServiceGraph Builder (trace processing)
kubectl logs -l app.kubernetes.io/component=servicegraph-builder -n tracing

# OTel Collector (trace forwarding)
kubectl logs -l app.kubernetes.io/component=otel-collector -n tracing

# Beyla (trace collection)
kubectl logs -l app.kubernetes.io/component=beyla -n tracing

# Microservices
kubectl logs -l app=gateway -n simple-microservices
```

### Test Trace Flow
```bash
# Generate traffic
for i in {1..5}; do
  curl -X POST http://localhost:8080/users \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"User $i\",\"email\":\"user$i@example.com\"}"
  sleep 1
done

# Count processed traces
kubectl logs -l app.kubernetes.io/component=servicegraph-builder -n tracing | grep "Converted SimpleSpan" | wc -l
```

## 🧹 Cleanup

### Remove Everything
```bash
./quick-start.sh cleanup
```

### Manual Cleanup
```bash
# Remove microservices
kubectl delete -f simple-microservices/k8s/ --namespace simple-microservices

# Remove tracing stack
helm uninstall tracing-demo --namespace tracing

# Stop port forwarding
pkill -f "kubectl port-forward"
```

## 🐛 Troubleshooting

### Common Issues

1. **Images not found in minikube**
   ```bash
   # Reload images
   minikube image load servicegraph-builder:v2
   ```

2. **Pods not starting**
   ```bash
   # Check pod status
   kubectl describe pod <pod-name> -n <namespace>
   ```

3. **No traces appearing**
   ```bash
   # Check OTel Collector logs for errors
   kubectl logs -l app.kubernetes.io/component=otel-collector -n tracing
   ```

4. **Port forward fails**
   ```bash
   # Kill existing port forwards
   pkill -f "kubectl port-forward"
   # Restart
   kubectl port-forward service/gateway 8080:8080 --namespace simple-microservices
   ```

### Debug Commands
```bash
# Check all resources
kubectl get all -n tracing
kubectl get all -n simple-microservices

# Check events
kubectl get events -n tracing --sort-by='.lastTimestamp'

# Check service endpoints
kubectl get endpoints -n tracing
```

## 📊 Expected Results

When working correctly, you should see:

1. **All pods running**: `kubectl get pods -n tracing` shows all pods as `Running`
2. **Trace processing logs**: ServiceGraph Builder logs show "Converted SimpleSpan" entries
3. **HTTP responses**: Curl commands return 200/201 status codes
4. **Service communication**: Traces show Gateway → User Service → Data Service flow

## 🎯 Success Indicators

- ✅ ServiceGraph Builder logs show "Converted SimpleSpan" messages
- ✅ HTTP requests to microservices return successful responses
- ✅ Beyla logs show instrumentation of microservices processes
- ✅ OTel Collector logs show trace export without errors
