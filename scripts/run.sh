#!/bin/bash

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
    docker-compose down
    
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

# Start Neo4j and frontend using docker-compose
echo "Starting Neo4j and frontend..."
docker-compose up -d

# Wait for Neo4j to be ready
echo "Waiting for Neo4j to be ready..."
until curl -s http://localhost:7474 > /dev/null; do
    echo "Waiting for Neo4j..."
    sleep 2
done
echo "Neo4j is ready!"

# Wait for frontend to be ready
echo "Waiting for frontend to be ready..."
until curl -s http://localhost:3000 > /dev/null; do
    echo "Waiting for frontend..."
    sleep 2
done
echo "Frontend is ready!"

# Start minikube
echo "Starting minikube..."
minikube start --driver=docker

# Run the application
echo "Starting application with skaffold..."
skaffold dev

# Keep the script running until Ctrl+C
wait
