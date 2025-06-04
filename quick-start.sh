#!/bin/bash

# Quick Start Script for Distributed Tracing Demo
# Simplified commands for common operations

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

case "${1:-help}" in
    "deploy")
        print_step "🚀 Quick Deploy - Full Stack"
        ./deploy-tracing-stack.sh
        ;;
    
    "build")
        print_step "🔨 Building Docker Images"
        cd servicegraph-builder && ./build.sh docker && cd ..
        cd simple-microservices && make docker-build && cd ..
        minikube image load servicegraph-builder:v2
        minikube image load simple-microservices/gateway:latest
        minikube image load simple-microservices/user-service:latest
        minikube image load simple-microservices/data-service:latest
        print_status "✅ Images built and loaded into minikube"
        ;;
    
    "test")
        print_step "🧪 Testing Trace Flow"
        kubectl port-forward service/gateway 8080:8080 --namespace simple-microservices &
        sleep 3
        
        for i in {1..3}; do
            curl -X POST http://localhost:8080/users \
                -H "Content-Type: application/json" \
                -d "{\"name\":\"Test User $i\",\"email\":\"test$i@example.com\"}" \
                -w "\nStatus: %{http_code}\n"
            sleep 1
        done
        
        sleep 5
        print_status "Checking traces..."
        kubectl logs -l app.kubernetes.io/component=servicegraph-builder -n tracing | grep "Converted SimpleSpan" | wc -l
        ;;
    
    "logs")
        print_step "📋 Viewing Component Logs"
        echo "=== ServiceGraph Builder Logs ==="
        kubectl logs -l app.kubernetes.io/component=servicegraph-builder -n tracing --tail=10
        echo ""
        echo "=== OTel Collector Logs ==="
        kubectl logs -l app.kubernetes.io/component=otel-collector -n tracing --tail=10
        echo ""
        echo "=== Beyla Logs ==="
        kubectl logs -l app.kubernetes.io/component=beyla -n tracing --tail=10
        ;;
    
    "status")
        print_step "📊 Component Status"
        echo "=== Tracing Components ==="
        kubectl get pods -n tracing
        echo ""
        echo "=== Microservices ==="
        kubectl get pods -n simple-microservices
        ;;
    
    "cleanup")
        print_step "🧹 Cleaning Up"
        kubectl delete -f simple-microservices/k8s/ --namespace simple-microservices || true
        helm uninstall tracing-demo --namespace tracing || true
        pkill -f "kubectl port-forward" || true
        print_status "✅ Cleanup complete"
        ;;
    
    "restart")
        print_step "🔄 Restarting Components"
        kubectl rollout restart deployment -n tracing
        kubectl rollout restart deployment -n simple-microservices
        print_status "✅ Components restarted"
        ;;
    
    "port-forward")
        print_step "🌐 Setting up Port Forward"
        print_status "Gateway will be available at http://localhost:8080"
        kubectl port-forward service/gateway 8080:8080 --namespace simple-microservices
        ;;
    
    *)
        cat << EOF
🚀 Distributed Tracing Quick Start

Usage: $0 [COMMAND]

Commands:
    deploy          Deploy the complete tracing stack
    build           Build and load Docker images
    test            Generate test traffic and check traces
    logs            View component logs
    status          Show component status
    cleanup         Remove all deployed resources
    restart         Restart all components
    port-forward    Set up port forwarding to gateway

Examples:
    $0 deploy       # Full deployment
    $0 test         # Quick test
    $0 logs         # View logs
    $0 cleanup      # Clean up everything

Full deployment process:
1. $0 deploy        # Deploys everything and runs tests
2. $0 test          # Generate more test traffic
3. $0 logs          # Check trace processing
4. $0 cleanup       # Clean up when done
EOF
        ;;
esac
