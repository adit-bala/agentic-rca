.PHONY: build clean run-local stop-local deploy undeploy test docker-build

# Build all services
build:
	@echo "Building all services..."
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/user-service ./cmd/user-service
	go build -o bin/data-service ./cmd/data-service
	@echo "Build complete!"

# Clean build artifacts
clean:
	rm -rf bin/
	@echo "Cleaned build artifacts"

# Run all services locally
run-local: build
	@echo "Starting all services locally..."
	@echo "Starting Data Service on port 8082..."
	./bin/data-service &
	@sleep 2
	@echo "Starting User Service on port 8081..."
	./bin/user-service &
	@sleep 2
	@echo "Starting API Gateway on port 8080..."
	./bin/gateway &
	@sleep 2
	@echo "All services started!"
	@echo "API Gateway: http://localhost:8080"
	@echo "User Service: http://localhost:8081"
	@echo "Data Service: http://localhost:8082"

# Stop local services
stop-local:
	@echo "Stopping all services..."
	pkill -f "./bin/" || true
	@echo "All services stopped"

# Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker build -t simple-microservices/gateway -f docker/Dockerfile.gateway .
	docker build -t simple-microservices/user-service -f docker/Dockerfile.user-service .
	docker build -t simple-microservices/data-service -f docker/Dockerfile.data-service .
	@echo "Docker images built!"

# Deploy to Kubernetes
deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -f k8s/
	@echo "Deployment complete!"
	@echo "Waiting for services to be ready..."
	kubectl wait --for=condition=ready pod -l app=gateway --timeout=60s
	kubectl wait --for=condition=ready pod -l app=user-service --timeout=60s
	kubectl wait --for=condition=ready pod -l app=data-service --timeout=60s
	@echo "All services are ready!"

# Remove from Kubernetes
undeploy:
	@echo "Removing from Kubernetes..."
	kubectl delete -f k8s/ || true
	@echo "Cleanup complete!"

# Test the service chain
test:
	@echo "Testing service chain..."
	@echo "Creating a user..."
	curl -X POST http://localhost:8080/users \
		-H "Content-Type: application/json" \
		-d '{"name":"Test User","email":"test@example.com"}' \
		-w "\nStatus: %{http_code}\n"
	@echo ""
	@echo "Getting user by ID..."
	curl -X GET http://localhost:8080/users/1 -w "\nStatus: %{http_code}\n"
	@echo ""
	@echo "Getting all users..."
	curl -X GET http://localhost:8080/users -w "\nStatus: %{http_code}\n"
	@echo ""
	@echo "Test complete!"

# Test Kubernetes deployment
test-k8s:
	@echo "Testing Kubernetes deployment..."
	@echo "Port-forwarding gateway service..."
	kubectl port-forward service/gateway 8080:8080 &
	@sleep 3
	@echo "Creating a user..."
	curl -X POST http://localhost:8080/users \
		-H "Content-Type: application/json" \
		-d '{"name":"K8s Test User","email":"k8s@example.com"}' \
		-w "\nStatus: %{http_code}\n"
	@echo "Stopping port-forward..."
	pkill -f "kubectl port-forward" || true
	@echo "Kubernetes test complete!"

# Show service logs (for debugging)
logs:
	@echo "=== Gateway Logs ==="
	kubectl logs -l app=gateway --tail=20
	@echo ""
	@echo "=== User Service Logs ==="
	kubectl logs -l app=user-service --tail=20
	@echo ""
	@echo "=== Data Service Logs ==="
	kubectl logs -l app=data-service --tail=20
