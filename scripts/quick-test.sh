#!/bin/bash

# Quick test script for the distributed tracing demo
# This script sets up port-forwarding and runs test commands

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

NAMESPACE="simple-microservices"
GATEWAY_URL="http://localhost:8080"

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Check if services are deployed
check_deployment() {
    print_info "Checking if services are deployed..."
    
    if ! kubectl get namespace $NAMESPACE &> /dev/null; then
        echo "Error: Namespace $NAMESPACE not found. Please deploy first:"
        echo "  ./deploy-tracing-demo.sh deploy"
        exit 1
    fi
    
    if ! kubectl get service gateway -n $NAMESPACE &> /dev/null; then
        echo "Error: Gateway service not found. Please deploy first:"
        echo "  ./deploy-tracing-demo.sh deploy"
        exit 1
    fi
    
    print_success "Services are deployed"
}

# Start port forwarding
start_port_forward() {
    print_info "Starting port-forward to gateway service..."
    
    # Kill any existing port-forward on 8080
    pkill -f "kubectl port-forward.*8080:8080" 2>/dev/null || true
    
    # Start new port-forward in background
    kubectl port-forward service/gateway 8080:8080 -n $NAMESPACE &
    PORT_FORWARD_PID=$!
    
    # Wait for port-forward to be ready
    print_info "Waiting for port-forward to be ready..."
    sleep 3
    
    # Test if port-forward is working
    if curl -s $GATEWAY_URL/health > /dev/null; then
        print_success "Port-forward is ready at $GATEWAY_URL"
    else
        echo "Error: Port-forward failed. Please check manually:"
        echo "  kubectl port-forward service/gateway 8080:8080 -n $NAMESPACE"
        exit 1
    fi
}

# Run test commands
run_tests() {
    print_header "Running Test Commands"
    
    # Test 1: Health check
    print_info "Testing health endpoint..."
    curl -s $GATEWAY_URL/health | jq . 2>/dev/null || curl -s $GATEWAY_URL/health
    echo ""
    
    # Test 2: Create users (generates traces)
    print_info "Creating test users (this generates distributed traces)..."
    
    for i in {1..5}; do
        echo "Creating user $i..."
        curl -X POST $GATEWAY_URL/users \
            -H "Content-Type: application/json" \
            -d "{\"name\":\"User $i\",\"email\":\"user$i@example.com\"}" \
            -w " (Status: %{http_code}, Time: %{time_total}s)\n" \
            -s -o /dev/null
        
        sleep 0.5
    done
    
    # Test 3: Get users
    print_info "Getting users..."
    curl -s $GATEWAY_URL/users/1 | jq . 2>/dev/null || curl -s $GATEWAY_URL/users/1
    echo ""
    
    # Test 4: List all users
    print_info "Listing all users..."
    curl -s $GATEWAY_URL/users | jq . 2>/dev/null || curl -s $GATEWAY_URL/users
    echo ""
    
    print_success "Test commands completed!"
}

# Generate continuous load
generate_load() {
    local duration=${1:-60}
    print_header "Generating Load for $duration seconds"
    
    print_info "Creating users continuously for $duration seconds..."
    print_info "This will generate many distributed traces for testing"
    
    local end_time=$((SECONDS + duration))
    local counter=1
    
    while [ $SECONDS -lt $end_time ]; do
        # Create user
        curl -X POST $GATEWAY_URL/users \
            -H "Content-Type: application/json" \
            -d "{\"name\":\"LoadTest User $counter\",\"email\":\"loadtest$counter@example.com\"}" \
            -s -o /dev/null
        
        # Occasionally get users
        if [ $((counter % 3)) -eq 0 ]; then
            curl -s $GATEWAY_URL/users/$((counter - 1)) -o /dev/null
        fi
        
        # Occasionally list users
        if [ $((counter % 5)) -eq 0 ]; then
            curl -s $GATEWAY_URL/users -o /dev/null
        fi
        
        echo -n "."
        counter=$((counter + 1))
        sleep 0.2
    done
    
    echo ""
    print_success "Generated $((counter - 1)) requests with distributed traces!"
}

# Show trace viewing instructions
show_trace_instructions() {
    print_header "Viewing Traces"
    
    cat << EOF
${YELLOW}To view the generated traces:${NC}

${BLUE}1. View OTel Collector logs (if using logging backend):${NC}
   kubectl logs -f deployment/servicegraph-demo-otel-collector -n tracing

${BLUE}2. View Beyla logs:${NC}
   kubectl logs -f daemonset/servicegraph-demo-beyla -n tracing

${BLUE}3. View service logs:${NC}
   kubectl logs -f deployment/gateway -n $NAMESPACE
   kubectl logs -f deployment/user-service -n $NAMESPACE
   kubectl logs -f deployment/data-service -n $NAMESPACE

${BLUE}4. If using external tracing backend:${NC}
   Check your Jaeger/Tempo/Zipkin UI for traces showing:
   - Gateway -> User Service -> Data Service call chain
   - Multiple spans with realistic timing
   - Service dependencies and performance metrics

${BLUE}5. Expected trace pattern:${NC}
   Each user creation should show:
   - Gateway: HTTP request handling (~50ms)
   - User Service: User creation + data processing call (~100ms)
   - Data Service: Multi-step data processing (~200-400ms)
EOF
}

# Cleanup function
cleanup() {
    print_info "Cleaning up port-forward..."
    pkill -f "kubectl port-forward.*8080:8080" 2>/dev/null || true
    print_success "Cleanup completed"
}

# Trap to cleanup on exit
trap cleanup EXIT

# Main execution
case "${1:-test}" in
    "test")
        check_deployment
        start_port_forward
        run_tests
        show_trace_instructions
        ;;
    "load")
        duration=${2:-60}
        check_deployment
        start_port_forward
        generate_load $duration
        show_trace_instructions
        ;;
    "health")
        check_deployment
        start_port_forward
        curl -s $GATEWAY_URL/health | jq . 2>/dev/null || curl -s $GATEWAY_URL/health
        ;;
    *)
        cat << EOF
Usage: $0 [COMMAND] [OPTIONS]

Commands:
    test        Run basic test commands (default)
    load [SEC]  Generate continuous load for SEC seconds (default: 60)
    health      Check service health

Examples:
    $0 test           # Run basic tests
    $0 load 120       # Generate load for 2 minutes
    $0 health         # Check health

This script will automatically set up port-forwarding and run tests.
Press Ctrl+C to stop and cleanup.
EOF
        ;;
esac
