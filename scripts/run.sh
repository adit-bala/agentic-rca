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
    
    # Stop and remove Neo4j container
    echo "Stopping Neo4j..."
    docker stop neo4j 2>/dev/null || true
    docker rm neo4j 2>/dev/null || true
    
    echo "Cleanup complete!"
    exit 0
}

# Set up trap for Ctrl+C
trap cleanup SIGINT SIGTERM

# Start Neo4j locally
echo "Starting Neo4j locally..."

docker run -d \
    --name neo4j \
    --network host \
    -e NEO4J_AUTH=neo4j/password \
    -e NEO4J_apoc_export_file_enabled=true \
    -e NEO4J_apoc_import_file_enabled=true \
    -e NEO4J_apoc_import_file_use__neo4j__config=true \
    neo4j:5.15.0

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

# Run the application
echo "Starting application with skaffold..."
skaffold dev
