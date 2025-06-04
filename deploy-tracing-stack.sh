#!/bin/bash

# Complete Distributed Tracing Stack Deployment Script
# This script builds, deploys, and tests the entire distributed tracing system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE_TRACING="tracing"
NAMESPACE_MICROSERVICES="simple-microservices"
RELEASE_NAME="tracing-demo"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Function to check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        print_error "$1 is not installed or not in PATH"
        exit 1
    fi
}

# Function to wait for pods to be ready
wait_for_pods() {
    local label="$1"
    local namespace="$2"
    local timeout="${3:-120}"
    
    print_status "Waiting for pods with label '$label' in namespace '$namespace' to be ready..."
    kubectl wait --for=condition=ready pod -l "$label" --namespace "$namespace" --timeout="${timeout}s" || {
        print_error "Pods failed to become ready within ${timeout} seconds"
        return 1
    }
}

# Function to cleanup on exit
cleanup() {
    print_status "Cleaning up port forwards..."
    pkill -f "kubectl port-forward.*8080:8080" 2>/dev/null || true
}

# Trap to cleanup on exit
trap cleanup EXIT

# Parse command line arguments
SKIP_BUILD=false
SKIP_DEPLOY=false
SKIP_TEST=false
CLEANUP_ON_EXIT=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-deploy)
            SKIP_DEPLOY=true
            shift
            ;;
        --skip-test)
            SKIP_TEST=true
            shift
            ;;
        --cleanup)
            CLEANUP_ON_EXIT=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --skip-build    Skip building Docker images"
            echo "  --skip-deploy   Skip deployment steps"
            echo "  --skip-test     Skip testing steps"
            echo "  --cleanup       Clean up resources after completion"
            echo "  --help, -h      Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

print_step "Starting Distributed Tracing Stack Deployment"
print_status "Configuration:"
print_status "  Tracing Namespace: $NAMESPACE_TRACING"
print_status "  Microservices Namespace: $NAMESPACE_MICROSERVICES"
print_status "  Helm Release: $RELEASE_NAME"

# Check prerequisites
print_step "Checking prerequisites..."
check_command "docker"
check_command "minikube"
check_command "kubectl"
check_command "helm"

# Check if minikube is running
if ! minikube status &> /dev/null; then
    print_status "Starting minikube..."
    minikube start --driver=docker
else
    print_status "Minikube is already running"
fi

# Build Docker images
if [ "$SKIP_BUILD" = false ]; then
    print_step "Building Docker images..."
    
    # Build servicegraph-builder
    print_status "Building servicegraph-builder..."
    cd "$SCRIPT_DIR/servicegraph-builder"
    ./build.sh docker
    
    # Build microservices
    print_status "Building microservices..."
    cd "$SCRIPT_DIR/simple-microservices"
    make docker-build
    
    cd "$SCRIPT_DIR"
    
    # Load images into minikube
    print_status "Loading images into minikube..."
    minikube image load servicegraph-builder:v2
    minikube image load simple-microservices/gateway:latest
    minikube image load simple-microservices/user-service:latest
    minikube image load simple-microservices/data-service:latest
else
    print_warning "Skipping Docker image builds"
fi

# Deploy the tracing stack
if [ "$SKIP_DEPLOY" = false ]; then
    print_step "Deploying distributed tracing stack..."
    
    # Deploy servicegraph-helm
    print_status "Deploying Helm chart (Beyla + OTel Collector + ServiceGraph Builder)..."
    helm upgrade --install "$RELEASE_NAME" ./servicegraph-helm \
        --namespace "$NAMESPACE_TRACING" \
        --create-namespace \
        --wait \
        --timeout=300s
    
    # Wait for tracing components to be ready
    wait_for_pods "app.kubernetes.io/component=servicegraph-builder" "$NAMESPACE_TRACING" 120
    wait_for_pods "app.kubernetes.io/component=otel-collector" "$NAMESPACE_TRACING" 120
    wait_for_pods "app.kubernetes.io/component=beyla" "$NAMESPACE_TRACING" 120
    
    # Deploy microservices
    print_status "Deploying microservices..."
    cd "$SCRIPT_DIR/simple-microservices"
    kubectl apply -f k8s/ --namespace="$NAMESPACE_MICROSERVICES"
    
    # Wait for microservices to be ready
    wait_for_pods "app=gateway" "$NAMESPACE_MICROSERVICES" 120
    wait_for_pods "app=user-service" "$NAMESPACE_MICROSERVICES" 120
    wait_for_pods "app=data-service" "$NAMESPACE_MICROSERVICES" 120
    
    cd "$SCRIPT_DIR"
else
    print_warning "Skipping deployment steps"
fi

# Test the system
if [ "$SKIP_TEST" = false ]; then
    print_step "Testing the distributed tracing system..."
    
    # Set up port forwarding
    print_status "Setting up port forwarding to gateway service..."
    kubectl port-forward service/gateway 8080:8080 --namespace "$NAMESPACE_MICROSERVICES" &
    PORT_FORWARD_PID=$!
    
    # Wait for port forward to be ready
    sleep 5
    
    # Test if port forward is working
    if ! curl -s http://localhost:8080/health > /dev/null; then
        print_error "Port forward failed. Gateway service may not be ready."
        exit 1
    fi
    
    print_status "✅ Port forward is ready at http://localhost:8080"
    
    # Generate test traffic
    print_status "Generating test traffic..."
    for i in {1..5}; do
        print_status "Creating test user $i..."
        curl -X POST http://localhost:8080/users \
            -H "Content-Type: application/json" \
            -d "{\"name\":\"Test User $i\",\"email\":\"test$i@example.com\"}" \
            -w "\nStatus: %{http_code}\n" \
            --max-time 10 || print_warning "Request $i failed"
        
        sleep 1
    done
    
    # Wait for traces to propagate
    print_status "Waiting for traces to propagate..."
    sleep 10
    
    # Check ServiceGraph Builder logs
    print_status "Checking ServiceGraph Builder logs for trace processing..."
    SERVICEGRAPH_POD=$(kubectl get pods -l app.kubernetes.io/component=servicegraph-builder --namespace="$NAMESPACE_TRACING" -o jsonpath='{.items[0].metadata.name}')
    
    if [ -z "$SERVICEGRAPH_POD" ]; then
        print_error "Could not find ServiceGraph Builder pod"
        exit 1
    fi
    
    print_status "ServiceGraph Builder pod: $SERVICEGRAPH_POD"
    
    # Count "Converted SimpleSpan" messages
    CONVERTED_SPANS=$(kubectl logs "$SERVICEGRAPH_POD" --namespace="$NAMESPACE_TRACING" | grep "Converted SimpleSpan" | wc -l)
    
    if [ "$CONVERTED_SPANS" -gt 0 ]; then
        print_status "✅ SUCCESS: Found $CONVERTED_SPANS 'Converted SimpleSpan' log entries!"
        print_status "Sample trace processing logs:"
        kubectl logs "$SERVICEGRAPH_POD" --namespace="$NAMESPACE_TRACING" | grep "Converted SimpleSpan" | head -3
    else
        print_error "❌ FAILURE: No 'Converted SimpleSpan' log entries found"
        print_status "ServiceGraph Builder logs:"
        kubectl logs "$SERVICEGRAPH_POD" --namespace="$NAMESPACE_TRACING" --tail=20
        exit 1
    fi
else
    print_warning "Skipping testing steps"
fi

# Show component status
print_step "Component status summary:"
print_status "Tracing components:"
kubectl get pods -l app.kubernetes.io/instance="$RELEASE_NAME" --namespace="$NAMESPACE_TRACING"
print_status "Microservices:"
kubectl get pods --namespace="$NAMESPACE_MICROSERVICES"

# Cleanup if requested
if [ "$CLEANUP_ON_EXIT" = true ]; then
    print_step "Cleaning up resources..."
    kubectl delete -f simple-microservices/k8s/ --namespace="$NAMESPACE_MICROSERVICES" || true
    helm uninstall "$RELEASE_NAME" --namespace="$NAMESPACE_TRACING" || true
    print_status "Cleanup complete"
fi

print_step "🎉 Distributed Tracing Stack Deployment Complete!"
print_status "✅ All components are running and processing traces"
print_status ""
print_status "Next steps:"
print_status "1. Access the gateway at: http://localhost:8080 (port-forward is active)"
print_status "2. View ServiceGraph Builder logs: kubectl logs -l app.kubernetes.io/component=servicegraph-builder -n $NAMESPACE_TRACING"
print_status "3. View Beyla logs: kubectl logs -l app.kubernetes.io/component=beyla -n $NAMESPACE_TRACING"
print_status "4. View OTel Collector logs: kubectl logs -l app.kubernetes.io/component=otel-collector -n $NAMESPACE_TRACING"
print_status ""
print_status "Press Ctrl+C to stop port forwarding and exit"

# Keep the script running to maintain port forward
if [ "$SKIP_TEST" = false ]; then
    wait
fi
