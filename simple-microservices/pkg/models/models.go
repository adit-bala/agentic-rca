package models

import "time"

// User represents a user in the system
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ProcessedData *ProcessedData `json:"processed_data,omitempty"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ProcessedData represents data processed by the data service
type ProcessedData struct {
	UserID      int       `json:"user_id"`
	ProcessedAt time.Time `json:"processed_at"`
	DataHash    string    `json:"data_hash"`
	Metrics     DataMetrics `json:"metrics"`
}

// DataMetrics represents metrics calculated during data processing
type DataMetrics struct {
	ProcessingTimeMs int     `json:"processing_time_ms"`
	DataSize         int     `json:"data_size"`
	Complexity       float64 `json:"complexity"`
}

// ProcessDataRequest represents a request to process user data
type ProcessDataRequest struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Service string `json:"service"`
}
