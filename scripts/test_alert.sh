#!/bin/bash

# Alert payload as a JSON string
ALERT_PAYLOAD='{
  "version": "4",
  "groupKey": "{}:{alertname=\"HighErrorRate\"}",
  "status": "firing",
  "receiver": "webhook-receiver",
  "groupLabels": {
    "alertname": "HighErrorRate",
    "severity": "critical"
  },
  "commonLabels": {
    "alertname": "HighErrorRate",
    "severity": "critical",
    "service": "data-service",
    "type": "invalid_request",
    "namespace": "simple-microservices",
    "pod": "data-service-7d8f9b6c5-4x3y2"
  },
  "commonAnnotations": {
    "summary": "High error rate detected",
    "description": "Service data-service has error rate above 5% for error type invalid_request"
  },
  "externalURL": "http://alertmanager:9093",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighErrorRate",
        "severity": "critical",
        "service": "data-service",
        "type": "invalid_request",
        "namespace": "simple-microservices",
        "pod": "data-service-7d8f9b6c5-4x3y2"
      },
      "annotations": {
        "summary": "High error rate detected",
        "description": "Service data-service has error rate above 5% for error type invalid_request"
      },
      "startsAt": "2024-03-20T14:20:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://prometheus:9090/graph?g0.expr=sum%28rate%28errors_total%5B5m%5D%29%29+by+%28service%2C+type%29+%2F+sum%28rate%28api_requests_total%5B5m%5D%29%29+by+%28service%29+%3E+0.05&g0.tab=1"
    }
  ]
}'

# Function to test the alert
test_alert() {
    local send_to_frontend=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --frontend)
                send_to_frontend=true
                shift
                ;;
            *)
                echo "Unknown option: $1"
                echo "Usage: $0 [--frontend]"
                exit 1
                ;;
        esac
    done

    if [ "$send_to_frontend" = true ]; then
        # Send only to frontend
        echo -e "\nSending alert to frontend..."
        frontend_response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$ALERT_PAYLOAD" \
            http://localhost:3000/api/alerts/webhook)

        # Check if frontend curl command was successful
        if [ $? -eq 0 ]; then
            echo -e "\nAlert sent successfully to frontend!"
            echo -e "\nFrontend Response:"
            echo "$frontend_response" | jq '.' 2>/dev/null || echo "$frontend_response"
        else
            echo -e "\nError sending alert to frontend: Failed to connect to the service"
        fi
    else
        # Send to backend to get websocket ID
        echo "Sending alert to agents service..."
        backend_response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$ALERT_PAYLOAD" \
            http://localhost:8001/alerts)

        # Check if backend curl command was successful
        if [ $? -eq 0 ]; then
            echo -e "\nAlert sent successfully to backend!"
            echo -e "\nBackend Response:"
            echo "$backend_response" | jq '.' 2>/dev/null || echo "$backend_response"
            
            # Extract websocket ID from response
            ws_id=$(echo "$backend_response" | jq -r '.websocket_id')
            if [ -z "$ws_id" ]; then
                echo "Error: Could not extract websocket ID from response"
                exit 1
            fi
            
            echo -e "\nConnecting to WebSocket for RCA updates..."
            
            # Use websocat to connect to the WebSocket and print updates
            # Note: You'll need to install websocat first (e.g., via cargo install websocat)
            websocat "ws://localhost:8001/process/$ws_id" | while read -r line; do
                echo "Received update:"
                echo "$line" | jq '.' 2>/dev/null || echo "$line"
            done
            
        else
            echo -e "\nError sending alert to backend: Failed to connect to the service"
        fi
    fi
}

# Run the test with any provided arguments
test_alert "$@" 