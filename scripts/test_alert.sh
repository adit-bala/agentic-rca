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
    # Send the alert using curl
    echo "Sending alert to agents service..."
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "$ALERT_PAYLOAD" \
        http://localhost:8001/alerts)

    # Check if curl command was successful
    if [ $? -eq 0 ]; then
        echo -e "\nAlert sent successfully!"
        echo -e "\nResponse:"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        echo -e "\nError sending alert: Failed to connect to the service"
    fi
}

# Run the test
test_alert 