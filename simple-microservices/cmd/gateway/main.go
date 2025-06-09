package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	httpClient "simple-microservices/pkg/http"
	"simple-microservices/pkg/metrics"
	"simple-microservices/pkg/models"
)

type Gateway struct {
	userServiceClient *httpClient.Client
	startTime         time.Time
}

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Start metrics server
	metrics.Start(log.Logger)

	// Get service URLs from environment or use defaults
	userServiceURL := getEnv("USER_SERVICE_URL", "http://localhost:8081")

	gateway := &Gateway{
		userServiceClient: httpClient.NewClient(userServiceURL),
		startTime:         time.Now(),
	}

	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", gateway.healthHandler).Methods("GET")

	// User routes (proxy to user service)
	r.HandleFunc("/users", gateway.createUserHandler).Methods("POST")
	r.HandleFunc("/users/{id}", gateway.getUserHandler).Methods("GET")
	r.HandleFunc("/users", gateway.listUsersHandler).Methods("GET")

	// Add middleware
	r.Use(loggingMiddleware)
	r.Use(tracingMiddleware)

	port := getEnv("PORT", "8080")
	log.Info().Str("port", port).Str("user_service", userServiceURL).Msg("API Gateway starting")

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Fatal().Err(server.ListenAndServe()).Msg("Server failed")
}

func (g *Gateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := models.HealthResponse{
		Status:    "healthy",
		Service:   "gateway",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Uptime:    time.Since(g.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (g *Gateway) createUserHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Invalid request body")
		metrics.Inc(metrics.ErrorTotal, prometheus.Labels{
			"service": "gateway",
			"type":    "invalid_request",
		}, 1)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	log.Info().Str("name", req.Name).Str("email", req.Email).Msg("Creating user via gateway")

	// Add artificial delay to make traces more interesting
	time.Sleep(50 * time.Millisecond)

	// Forward to user service
	var user models.User
	if err := g.userServiceClient.Post(ctx, "/users", req, &user); err != nil {
		log.Error().Err(err).Msg("Failed to create user")
		metrics.Inc(metrics.ErrorTotal, prometheus.Labels{
			"service": "gateway",
			"type":    "user_service_error",
		}, 1)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	log.Info().Int("user_id", user.ID).Str("status", user.Status).Msg("User created successfully")

	// Record metrics
	metrics.Observe(metrics.APIRequestLatency, prometheus.Labels{
		"service":  "gateway",
		"endpoint": "/users",
		"method":   "POST",
	}, time.Since(start).Seconds())

	metrics.Inc(metrics.APIRequestTotal, prometheus.Labels{
		"service":  "gateway",
		"endpoint": "/users",
		"method":   "POST",
		"status":   "success",
	}, 1)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (g *Gateway) getUserHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	userID := vars["id"]

	ctx := r.Context()

	log.Info().Str("user_id", userID).Msg("Getting user via gateway")

	var user models.User
	if err := g.userServiceClient.Get(ctx, "/users/"+userID, &user); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		metrics.Inc(metrics.ErrorTotal, prometheus.Labels{
			"service": "gateway",
			"type":    "user_not_found",
		}, 1)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Record metrics
	metrics.Observe(metrics.APIRequestLatency, prometheus.Labels{
		"service":  "gateway",
		"endpoint": "/users/{id}",
		"method":   "GET",
	}, time.Since(start).Seconds())

	metrics.Inc(metrics.APIRequestTotal, prometheus.Labels{
		"service":  "gateway",
		"endpoint": "/users/{id}",
		"method":   "GET",
		"status":   "success",
	}, 1)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (g *Gateway) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	log.Info().Msg("Listing users via gateway")

	var users []models.User
	if err := g.userServiceClient.Get(ctx, "/users", &users); err != nil {
		log.Error().Err(err).Msg("Failed to list users")
		metrics.Inc(metrics.ErrorTotal, prometheus.Labels{
			"service": "gateway",
			"type":    "list_users_error",
		}, 1)
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	// Record metrics
	metrics.Observe(metrics.APIRequestLatency, prometheus.Labels{
		"service":  "gateway",
		"endpoint": "/users",
		"method":   "GET",
	}, time.Since(start).Seconds())

	metrics.Inc(metrics.APIRequestTotal, prometheus.Labels{
		"service":  "gateway",
		"endpoint": "/users",
		"method":   "GET",
		"status":   "success",
	}, 1)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Str("service", "gateway").
			Msg("Request started")

		next.ServeHTTP(w, r)

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("service", "gateway").
			Dur("duration", time.Since(start)).
			Msg("Request completed")
	})
}

func tracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate or forward trace ID
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = fmt.Sprintf("trace-%d", time.Now().UnixNano())
		}

		ctx := context.WithValue(r.Context(), "trace-id", traceID)
		w.Header().Set("X-Trace-ID", traceID)

		log.Info().Str("trace_id", traceID).Msg("Processing request with trace ID")

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
