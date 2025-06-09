package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"simple-microservices/pkg/metrics"
	"simple-microservices/pkg/models"
)

type DataService struct {
	startTime time.Time
}

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Start metrics server
	metrics.Start(log.Logger)

	service := &DataService{
		startTime: time.Now(),
	}

	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", service.healthHandler).Methods("GET")

	// Data processing routes
	r.HandleFunc("/process", service.processDataHandler).Methods("POST")
	r.HandleFunc("/status", service.statusHandler).Methods("GET")

	// Add middleware
	r.Use(loggingMiddleware)

	port := getEnv("PORT", "8082")
	log.Info().Str("port", port).Msg("Data Service starting")

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Fatal().Err(server.ListenAndServe()).Msg("Server failed")
}

func (s *DataService) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := models.HealthResponse{
		Status:    "healthy",
		Service:   "data-service",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Uptime:    time.Since(s.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *DataService) processDataHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req models.ProcessDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Invalid request body")
		metrics.Inc(metrics.ErrorTotal, prometheus.Labels{
			"service": "data-service",
			"type":    "invalid_request",
		}, 1)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	traceID := r.Header.Get("X-Trace-ID")
	log.Info().
		Int("user_id", req.UserID).
		Str("name", req.Name).
		Str("email", req.Email).
		Str("trace_id", traceID).
		Msg("Processing user data")

	// Simulate complex data processing with multiple steps
	log.Info().Int("user_id", req.UserID).Msg("Step 1: Validating data")
	time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)

	log.Info().Int("user_id", req.UserID).Msg("Step 2: Computing hash")
	dataHash := s.computeDataHash(req.Name, req.Email)
	time.Sleep(time.Duration(rand.Intn(75)+25) * time.Millisecond)

	log.Info().Int("user_id", req.UserID).Msg("Step 3: Calculating metrics")
	complexity := s.calculateMetrics(req.Name, req.Email)
	time.Sleep(time.Duration(rand.Intn(150)+100) * time.Millisecond)

	log.Info().Int("user_id", req.UserID).Msg("Step 4: Finalizing processing")
	time.Sleep(time.Duration(rand.Intn(50)+25) * time.Millisecond)

	processingTime := time.Since(start)

	processedData := models.ProcessedData{
		UserID:      req.UserID,
		ProcessedAt: time.Now(),
		DataHash:    dataHash,
		Metrics: models.DataMetrics{
			ProcessingTimeMs: int(processingTime.Milliseconds()),
			DataSize:         len(req.Name) + len(req.Email),
			Complexity:       complexity,
		},
	}

	// Record metrics
	metrics.Observe(metrics.APIRequestLatency, prometheus.Labels{
		"service":  "data-service",
		"endpoint": "/process",
		"method":   "POST",
	}, time.Since(start).Seconds())

	metrics.Inc(metrics.APIRequestTotal, prometheus.Labels{
		"service":  "data-service",
		"endpoint": "/process",
		"method":   "POST",
		"status":   "success",
	}, 1)

	metrics.Set(metrics.ActiveConnections, prometheus.Labels{
		"service": "data-service",
	}, float64(1)) // Each processing request is an active connection

	log.Info().
		Int("user_id", req.UserID).
		Dur("processing_time", processingTime).
		Str("data_hash", dataHash).
		Float64("complexity", complexity).
		Msg("Data processing completed")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(processedData)
}

func (s *DataService) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":            "data-service",
		"status":             "healthy",
		"uptime":             time.Since(s.startTime).String(),
		"processed_requests": rand.Intn(1000) + 100,
		"avg_processing_ms":  rand.Intn(200) + 50,
		"cache_hit_ratio":    0.75 + rand.Float64()*0.2,
		"memory_usage_mb":    rand.Intn(100) + 50,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *DataService) computeDataHash(name, email string) string {
	data := fmt.Sprintf("%s:%s:%d", name, email, time.Now().UnixNano())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)[:16] // Return first 16 characters
}

func (s *DataService) calculateMetrics(name, email string) float64 {
	// Simulate complexity calculation based on data characteristics
	nameComplexity := float64(len(name)) * 0.1
	emailComplexity := float64(len(email)) * 0.15
	randomFactor := rand.Float64() * 2.0

	complexity := nameComplexity + emailComplexity + randomFactor
	return complexity
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID := r.Header.Get("X-Trace-ID")

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("service", "data-service").
			Str("trace_id", traceID).
			Msg("Request started")

		next.ServeHTTP(w, r)

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("service", "data-service").
			Str("trace_id", traceID).
			Dur("duration", time.Since(start)).
			Msg("Request completed")
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
