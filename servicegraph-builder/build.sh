#!/bin/bash

# Simple build script for servicegraph-builder
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# Default values
ACTION="build"
DOCKER_BUILD=false
RUN_LOCAL=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        build)
            ACTION="build"
            shift
            ;;
        run)
            ACTION="run"
            shift
            ;;
        docker)
            DOCKER_BUILD=true
            shift
            ;;
        clean)
            ACTION="clean"
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [build|run|docker|clean] [options]"
            echo ""
            echo "Commands:"
            echo "  build     Build the servicegraph-builder binary (default)"
            echo "  run       Build and run the servicegraph-builder locally"
            echo "  docker    Build Docker image"
            echo "  clean     Clean build artifacts"
            echo ""
            echo "Examples:"
            echo "  $0 build          # Build binary"
            echo "  $0 run            # Build and run locally"
            echo "  $0 docker         # Build Docker image"
            echo "  $0 clean          # Clean artifacts"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Ensure we have Go installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

case $ACTION in
    build)
        print_status "Building servicegraph-builder..."
        go mod tidy
        go build -o bin/servicegraph-builder ./cmd/server
        print_status "Build complete! Binary: bin/servicegraph-builder"
        ;;
    
    run)
        print_status "Building and running servicegraph-builder..."
        go mod tidy
        go build -o bin/servicegraph-builder ./cmd/server
        print_status "Starting servicegraph-builder on localhost:8083..."
        print_warning "Press Ctrl+C to stop"
        ./bin/servicegraph-builder
        ;;
    
    clean)
        print_status "Cleaning build artifacts..."
        rm -rf bin/
        print_status "Clean complete!"
        ;;
esac

if $DOCKER_BUILD; then
    print_status "Building Docker image..."
    docker build -t servicegraph-builder:latest -f docker/Dockerfile.servicegraph .
    print_status "Docker image built: servicegraph-builder:latest"
fi
