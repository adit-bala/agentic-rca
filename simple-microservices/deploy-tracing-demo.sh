#!/bin/bash

# Distributed Tracing Demo Deployment Script
# Deploys servicegraph-helm chart and simple-microservices for testing Beyla tracing

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
MICROSERVICES_NAMESPACE="simple-microservices"
SERVICEGRAPH_NAMESPACE="tracing"
SERVICEGRAPH_RELEASE="servicegraph-demo"
SERVICEGRAPH_CHART_PATH="../servicegraph-helm"

# Default tracing backend (can be overridden)
TRACING_BACKEND=${TRACING_BACKEND:-"logging"}
TRACING_ENDPOINT=${TRACING_ENDPOINT:-""}
FAST_MODE=false

# Print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Show usage
show_usage() {
    cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

Commands:
    deploy      Deploy both servicegraph and microservices (default)
    cleanup     Remove all deployments
    status      Show deployment status
    test        Run test commands
    logs        Show logs from all services

Options:
    --backend BACKEND    Tracing backend: logging (default), jaeger, tempo, zipkin
    --endpoint URL       Tracing backend endpoint (e.g., http://jaeger:14250)
    --fast              Skip Docker builds if images exist (faster deployment)
    --help              Show this help message

Examples:
    $0 deploy                                    # Deploy with logging backend
    $0 deploy --fast                             # Fast deploy (skip builds if possible)
    $0 deploy --backend jaeger --endpoint http://jaeger:14250
    $0 cleanup                                   # Remove all deployments
    $0 test                                      # Run test commands

Environment Variables:
    TRACING_BACKEND     Override default tracing backend
    TRACING_ENDPOINT    Override default tracing endpoint
EOF
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    # Check if minikube is installed and running
    if ! command -v minikube &> /dev/null; then
        print_error "minikube is not installed. Please install minikube first."
        exit 1
    fi
    
    if ! minikube status &> /dev/null; then
        print_error "minikube is not running. Please start minikube first:"
        echo "  minikube start"
        exit 1
    fi
    print_success "minikube is running"
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi
    print_success "kubectl is available"
    
    # Check helm
    if ! command -v helm &> /dev/null; then
        print_error "helm is not installed. Please install helm first."
        exit 1
    fi
    print_success "helm is available"
    
    # Check if servicegraph chart exists
    if [ ! -d "$SERVICEGRAPH_CHART_PATH" ]; then
        print_error "servicegraph-helm chart not found at $SERVICEGRAPH_CHART_PATH"
        print_info "Please ensure the servicegraph-helm directory exists relative to this script"
        exit 1
    fi
    print_success "servicegraph-helm chart found"
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "docker is not installed. Please install docker first."
        exit 1
    fi
    print_success "docker is available"
    
    print_success "All prerequisites met!"
}

# Setup minikube Docker environment
setup_docker_env() {
    print_header "Setting up Minikube Docker Environment"
    
    print_info "Configuring Docker to use minikube's Docker daemon..."
    eval $(minikube docker-env)
    
    print_success "Docker environment configured for minikube"
}

# Build Docker images
build_images() {
    print_header "Building Docker Images"

    # Fast mode: skip if images exist
    if [ "$FAST_MODE" = true ]; then
        if docker images | grep -q "simple-microservices/gateway" && \
           docker images | grep -q "simple-microservices/user-service" && \
           docker images | grep -q "simple-microservices/data-service"; then
            print_success "Images already exist, skipping build (fast mode)"
            return
        fi
    fi

    print_info "Building microservices Docker images with correct architecture..."

    # Build images using minikube's Docker daemon
    print_info "Building gateway image..."
    docker build -t simple-microservices/gateway:latest -f docker/Dockerfile.gateway .

    print_info "Building user-service image..."
    docker build -t simple-microservices/user-service:latest -f docker/Dockerfile.user-service .

    print_info "Building data-service image..."
    docker build -t simple-microservices/data-service:latest -f docker/Dockerfile.data-service .

    print_success "All Docker images built successfully"

    # List built images
    print_info "Built images:"
    docker images | grep simple-microservices
}

# Deploy microservices
deploy_microservices() {
    print_header "Deploying Simple Microservices"
    
    print_info "Creating namespace: $MICROSERVICES_NAMESPACE"
    kubectl create namespace $MICROSERVICES_NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    
    print_info "Deploying microservices..."
    kubectl apply -f k8s/
    
    print_info "Waiting for microservices to be ready..."
    kubectl wait --for=condition=ready pod -l app=data-service -n $MICROSERVICES_NAMESPACE --timeout=120s
    kubectl wait --for=condition=ready pod -l app=user-service -n $MICROSERVICES_NAMESPACE --timeout=120s
    kubectl wait --for=condition=ready pod -l app=gateway -n $MICROSERVICES_NAMESPACE --timeout=120s
    
    print_success "Microservices deployed and ready!"
}

# Deploy servicegraph
deploy_servicegraph() {
    print_header "Deploying ServiceGraph (Beyla + OTel Collector)"
    
    print_info "Creating namespace: $SERVICEGRAPH_NAMESPACE"
    kubectl create namespace $SERVICEGRAPH_NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    
    # Prepare helm values based on backend
    local helm_args=""
    case $TRACING_BACKEND in
        "jaeger")
            if [ -z "$TRACING_ENDPOINT" ]; then
                print_error "Jaeger endpoint required. Use --endpoint option."
                exit 1
            fi
            helm_args="--set otelCollector.export.endpoint=$TRACING_ENDPOINT"
            ;;
        "tempo")
            if [ -z "$TRACING_ENDPOINT" ]; then
                print_error "Tempo endpoint required. Use --endpoint option."
                exit 1
            fi
            helm_args="--set otelCollector.export.endpoint=$TRACING_ENDPOINT"
            ;;
        "zipkin")
            if [ -z "$TRACING_ENDPOINT" ]; then
                print_error "Zipkin endpoint required. Use --endpoint option."
                exit 1
            fi
            helm_args="--set otelCollector.export.endpoint=$TRACING_ENDPOINT"
            ;;
        "logging"|*)
            print_info "Using logging exporter (traces will be logged to OTel Collector)"
            helm_args=""
            ;;
    esac
    
    print_info "Installing servicegraph helm chart..."
    print_info "Backend: $TRACING_BACKEND"
    if [ -n "$TRACING_ENDPOINT" ]; then
        print_info "Endpoint: $TRACING_ENDPOINT"
    fi
    
    helm upgrade --install $SERVICEGRAPH_RELEASE $SERVICEGRAPH_CHART_PATH \
        --namespace $SERVICEGRAPH_NAMESPACE \
        --set global.namespace=$SERVICEGRAPH_NAMESPACE \
        --set beyla.discovery.namespace=$MICROSERVICES_NAMESPACE \
        $helm_args \
        --wait --timeout=300s
    
    print_success "ServiceGraph deployed successfully!"
}

# Show deployment status
show_status() {
    print_header "Deployment Status"
    
    print_info "Microservices namespace ($MICROSERVICES_NAMESPACE):"
    kubectl get pods,services -n $MICROSERVICES_NAMESPACE
    
    echo ""
    print_info "ServiceGraph namespace ($SERVICEGRAPH_NAMESPACE):"
    kubectl get pods,services -n $SERVICEGRAPH_NAMESPACE
    
    echo ""
    print_info "Helm releases:"
    helm list -n $SERVICEGRAPH_NAMESPACE
}

# Cleanup deployments
cleanup() {
    print_header "Cleaning Up Deployments"
    
    print_info "Removing servicegraph helm release..."
    helm uninstall $SERVICEGRAPH_RELEASE -n $SERVICEGRAPH_NAMESPACE || true
    
    print_info "Removing microservices..."
    kubectl delete -f k8s/ || true
    
    print_info "Removing namespaces..."
    kubectl delete namespace $MICROSERVICES_NAMESPACE || true
    kubectl delete namespace $SERVICEGRAPH_NAMESPACE || true
    
    print_success "Cleanup completed!"
}

# Show test instructions
show_test_instructions() {
    print_header "Testing Instructions"
    
    cat << EOF
${GREEN}Your distributed tracing demo is ready!${NC}

${YELLOW}1. Port-forward the gateway service:${NC}
   kubectl port-forward service/gateway 8080:8080 -n $MICROSERVICES_NAMESPACE

${YELLOW}2. In another terminal, test the service chain:${NC}
   # Create a user (triggers full trace chain)
   curl -X POST http://localhost:8080/users \\
     -H "Content-Type: application/json" \\
     -d '{"name":"John Doe","email":"john@example.com"}'
   
   # Get user by ID
   curl http://localhost:8080/users/1
   
   # List all users
   curl http://localhost:8080/users

${YELLOW}3. Generate load for testing:${NC}
   ./scripts/test-load.sh http://localhost:8080 20

${YELLOW}4. View traces:${NC}
EOF

    case $TRACING_BACKEND in
        "logging")
            cat << EOF
   # View OTel Collector logs to see traces
   kubectl logs -f deployment/servicegraph-demo-otel-collector -n $SERVICEGRAPH_NAMESPACE
EOF
            ;;
        *)
            cat << EOF
   # Check your tracing backend at: $TRACING_ENDPOINT
   # Look for traces showing: Gateway -> User Service -> Data Service
EOF
            ;;
    esac

    cat << EOF

${YELLOW}5. View service logs:${NC}
   kubectl logs -f deployment/gateway -n $MICROSERVICES_NAMESPACE
   kubectl logs -f deployment/user-service -n $MICROSERVICES_NAMESPACE
   kubectl logs -f deployment/data-service -n $MICROSERVICES_NAMESPACE

${YELLOW}6. View Beyla logs:${NC}
   kubectl logs -f daemonset/servicegraph-demo-beyla -n $SERVICEGRAPH_NAMESPACE

${YELLOW}7. Cleanup when done:${NC}
   $0 cleanup
EOF
}

# Run test commands
run_tests() {
    print_header "Running Test Commands"
    
    print_info "Checking if gateway is accessible..."
    if ! kubectl get service gateway -n $MICROSERVICES_NAMESPACE &> /dev/null; then
        print_error "Gateway service not found. Please deploy first."
        exit 1
    fi
    
    print_info "Starting port-forward in background..."
    kubectl port-forward service/gateway 8080:8080 -n $MICROSERVICES_NAMESPACE &
    PORT_FORWARD_PID=$!
    
    # Wait for port-forward to be ready
    sleep 3
    
    print_info "Testing service endpoints..."
    
    # Test health endpoints
    curl -s http://localhost:8080/health | jq . || echo "Gateway health check"
    
    # Create a test user
    print_info "Creating test user..."
    curl -X POST http://localhost:8080/users \
        -H "Content-Type: application/json" \
        -d '{"name":"Test User","email":"test@example.com"}' \
        -w "\nStatus: %{http_code}\n"
    
    # Get the user
    print_info "Getting user..."
    curl -s http://localhost:8080/users/1 | jq . || echo "Get user request"
    
    # Stop port-forward
    kill $PORT_FORWARD_PID 2>/dev/null || true
    
    print_success "Test commands completed!"
}

# Show logs
show_logs() {
    print_header "Service Logs"
    
    echo -e "${YELLOW}=== Gateway Logs ===${NC}"
    kubectl logs --tail=20 deployment/gateway -n $MICROSERVICES_NAMESPACE
    
    echo -e "\n${YELLOW}=== User Service Logs ===${NC}"
    kubectl logs --tail=20 deployment/user-service -n $MICROSERVICES_NAMESPACE
    
    echo -e "\n${YELLOW}=== Data Service Logs ===${NC}"
    kubectl logs --tail=20 deployment/data-service -n $MICROSERVICES_NAMESPACE
    
    echo -e "\n${YELLOW}=== Beyla Logs ===${NC}"
    kubectl logs --tail=20 daemonset/servicegraph-demo-beyla -n $SERVICEGRAPH_NAMESPACE
    
    echo -e "\n${YELLOW}=== OTel Collector Logs ===${NC}"
    kubectl logs --tail=20 deployment/servicegraph-demo-otel-collector -n $SERVICEGRAPH_NAMESPACE
}

# Main deployment function
deploy() {
    check_prerequisites
    setup_docker_env
    build_images
    deploy_microservices
    deploy_servicegraph
    show_status
    show_test_instructions
}

# Parse command line arguments
COMMAND="deploy"
while [[ $# -gt 0 ]]; do
    case $1 in
        deploy|cleanup|status|test|logs)
            COMMAND=$1
            shift
            ;;
        --backend)
            TRACING_BACKEND="$2"
            shift 2
            ;;
        --endpoint)
            TRACING_ENDPOINT="$2"
            shift 2
            ;;
        --fast)
            FAST_MODE=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Execute command
case $COMMAND in
    deploy)
        deploy
        ;;
    cleanup)
        cleanup
        ;;
    status)
        show_status
        ;;
    test)
        run_tests
        ;;
    logs)
        show_logs
        ;;
    *)
        print_error "Unknown command: $COMMAND"
        show_usage
        exit 1
        ;;
esac
