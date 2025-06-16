#!/bin/bash

# Parse command line arguments
FRESH=false
for arg in "$@"; do
    case $arg in
        --fresh)
        FRESH=true
        shift
        ;;
    esac
done

# Function to cleanup resources
cleanup() {
    echo "Cleaning up resources..."
    
    # Stop skaffold if it's running
    if [ -n "$(ps aux | grep skaffold | grep -v grep)" ]; then
        echo "Stopping skaffold..."
        pkill -f skaffold
    fi
    
    # Stop minikube
    echo "Stopping minikube..."
    minikube stop
    
    # Stop docker-compose services
    echo "Stopping docker-compose services..."
    if [ "$FRESH" = true ]; then
        docker-compose down -v
    else
        docker-compose down
    fi
    
    echo "Cleanup complete!"
    exit 0
}

# Set up trap for Ctrl+C
trap cleanup SIGINT SIGTERM

# Check if required tools are installed
echo "Checking required tools..."
command -v docker >/dev/null 2>&1 || { echo "Docker is required but not installed. Aborting."; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "Docker Compose is required but not installed. Aborting."; exit 1; }
command -v minikube >/dev/null 2>&1 || { echo "Minikube is required but not installed. Aborting."; exit 1; }
command -v skaffold >/dev/null 2>&1 || { echo "Skaffold is required but not installed. Aborting."; exit 1; }
command -v helm >/dev/null 2>&1 || { echo "Helm is required but not installed. Aborting."; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "npm is required but not installed. Aborting."; exit 1; }

# Check for Observe token
if [ -z "$OBSERVE_TOKEN" ]; then
    echo "Error: OBSERVE_TOKEN environment variable is not set"
    echo "Please set it with: export OBSERVE_TOKEN=your_token_here"
    exit 1
fi

# Start Neo4j using docker-compose
echo "Starting Neo4j..."
if [ "$FRESH" = true ]; then
    echo "Starting with fresh database..."
    docker-compose down -v
fi
docker-compose up -d

# Wait for Neo4j to be ready
echo "Waiting for Neo4j to be ready..."
until curl -s http://localhost:7474 > /dev/null; do
    echo "Waiting for Neo4j..."
    sleep 2
done
echo "Neo4j is ready!"

# Start minikube
echo "Starting minikube..."
minikube start --driver=docker

# Setup Observe
echo "Setting up Observe..."
kubectl create namespace observe
kubectl -n observe create secret generic agent-credentials --from-literal=OBSERVE_TOKEN="$OBSERVE_TOKEN"

kubectl annotate secret agent-credentials -n observe \
  meta.helm.sh/release-name=observe-agent \
  meta.helm.sh/release-namespace=observe

kubectl label secret agent-credentials -n observe \
  app.kubernetes.io/managed-by=Helm

helm repo add observe https://observeinc.github.io/observe-agent-helm
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install observe-agent observe/agent -n observe \
--set observe.collectionEndpoint.value="https://119137983744.collect.observeinc.com/" \
--set cluster.name="observe-agent-monitored-cluster" \
--set node.containers.logs.enabled="true" \
--set application.prometheusScrape.enabled="false" \
--set node.forwarder.enabled="false" \
--set node.forwarder.metrics.outputFormat="otel"

# Create monitoring namespace
kubectl create namespace monitoring

# Apply Prometheus scrape configuration
echo "Applying Prometheus scrape configuration..."
kubectl apply -f ./simple-microservices/k8s/prometheus-scrape-config.yaml

# Apply AlertManager configuration
echo "Applying AlertManager configuration..."
kubectl apply -f ./simple-microservices/k8s/alertmanager-config.yaml

# Apply Prometheus rules
echo "Applying Prometheus rules..."
kubectl apply -f ./simple-microservices/k8s/prometheus-rules.yaml

# Install kube-prometheus-stack with our configurations
echo "Installing kube-prometheus-stack..."
helm install prometheus \
  prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  -f ./simple-microservices/k8s/prometheus-values.yaml

# Run the application
echo "Starting application with skaffold..."
skaffold dev

# Keep the script running until Ctrl+C
wait
