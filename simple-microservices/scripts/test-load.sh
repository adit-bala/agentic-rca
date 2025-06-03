#!/bin/bash

# Test script to generate load for tracing demonstration

GATEWAY_URL=${1:-"http://localhost:8080"}
NUM_REQUESTS=${2:-10}

echo "Generating load against $GATEWAY_URL with $NUM_REQUESTS requests..."

# Function to create a user
create_user() {
    local id=$1
    local name="User$id"
    local email="user$id@example.com"
    
    echo "Creating user: $name ($email)"
    curl -s -X POST "$GATEWAY_URL/users" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"$name\",\"email\":\"$email\"}" \
        -w "Status: %{http_code}, Time: %{time_total}s\n" \
        -o /dev/null
}

# Function to get a user
get_user() {
    local id=$1
    echo "Getting user: $id"
    curl -s -X GET "$GATEWAY_URL/users/$id" \
        -w "Status: %{http_code}, Time: %{time_total}s\n" \
        -o /dev/null
}

# Function to list all users
list_users() {
    echo "Listing all users"
    curl -s -X GET "$GATEWAY_URL/users" \
        -w "Status: %{http_code}, Time: %{time_total}s\n" \
        -o /dev/null
}

# Generate load
echo "Starting load generation..."

for i in $(seq 1 $NUM_REQUESTS); do
    echo "--- Request $i ---"
    
    # Create a user (this will generate the full trace chain)
    create_user $i
    
    # Small delay between requests
    sleep 0.5
    
    # Occasionally get a user or list users
    if [ $((i % 3)) -eq 0 ]; then
        get_user $((i - 1))
    fi
    
    if [ $((i % 5)) -eq 0 ]; then
        list_users
    fi
    
    echo ""
done

echo "Load generation complete!"
echo ""
echo "Summary:"
echo "- Created $NUM_REQUESTS users"
echo "- Made additional GET requests for variety"
echo "- Each user creation triggers: Gateway -> User Service -> Data Service"
echo ""
echo "Check your tracing backend for the generated traces!"
