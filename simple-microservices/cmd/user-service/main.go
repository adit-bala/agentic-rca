package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	httpClient "simple-microservices/pkg/http"
	"simple-microservices/pkg/models"
)

type UserService struct {
	users           map[int]*models.User
	usersMutex      sync.RWMutex
	nextID          int
	dataServiceClient *httpClient.Client
	startTime       time.Time
}

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Get service URLs from environment or use defaults
	dataServiceURL := getEnv("DATA_SERVICE_URL", "http://localhost:8082")

	service := &UserService{
		users:             make(map[int]*models.User),
		nextID:            1,
		dataServiceClient: httpClient.NewClient(dataServiceURL),
		startTime:         time.Now(),
	}

	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", service.healthHandler).Methods("GET")

	// User routes
	r.HandleFunc("/users", service.createUserHandler).Methods("POST")
	r.HandleFunc("/users/{id}", service.getUserHandler).Methods("GET")
	r.HandleFunc("/users", service.listUsersHandler).Methods("GET")

	// Add middleware
	r.Use(loggingMiddleware)

	port := getEnv("PORT", "8081")
	log.Info().Str("port", port).Str("data_service", dataServiceURL).Msg("User Service starting")

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Fatal().Err(server.ListenAndServe()).Msg("Server failed")
}

func (s *UserService) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := models.HealthResponse{
		Status:    "healthy",
		Service:   "user-service",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Uptime:    time.Since(s.startTime).String(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *UserService) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	log.Info().Str("name", req.Name).Str("email", req.Email).Msg("Creating user")

	// Add artificial delay for more interesting traces
	time.Sleep(100 * time.Millisecond)

	// Create user
	s.usersMutex.Lock()
	userID := s.nextID
	s.nextID++
	
	user := &models.User{
		ID:        userID,
		Name:      req.Name,
		Email:     req.Email,
		Status:    "processing",
		CreatedAt: time.Now(),
	}
	s.users[userID] = user
	s.usersMutex.Unlock()

	log.Info().Int("user_id", userID).Msg("User created, calling data service for processing")

	// Call data service to process user data
	processReq := models.ProcessDataRequest{
		UserID: userID,
		Name:   req.Name,
		Email:  req.Email,
	}

	var processedData models.ProcessedData
	if err := s.dataServiceClient.Post(ctx, "/process", processReq, &processedData); err != nil {
		log.Error().Err(err).Int("user_id", userID).Msg("Failed to process user data")
		user.Status = "error"
	} else {
		log.Info().Int("user_id", userID).Msg("User data processed successfully")
		user.Status = "active"
		user.ProcessedData = &processedData
	}

	log.Info().Int("user_id", userID).Str("status", user.Status).Msg("User creation completed")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *UserService) getUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDStr := vars["id"]
	
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Error().Err(err).Str("user_id", userIDStr).Msg("Invalid user ID")
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	log.Info().Int("user_id", userID).Msg("Getting user")

	// Add small delay
	time.Sleep(25 * time.Millisecond)

	s.usersMutex.RLock()
	user, exists := s.users[userID]
	s.usersMutex.RUnlock()

	if !exists {
		log.Warn().Int("user_id", userID).Msg("User not found")
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *UserService) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Listing all users")

	// Add small delay
	time.Sleep(50 * time.Millisecond)

	s.usersMutex.RLock()
	users := make([]*models.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	s.usersMutex.RUnlock()

	log.Info().Int("count", len(users)).Msg("Retrieved users")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID := r.Header.Get("X-Trace-ID")
		
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("service", "user-service").
			Str("trace_id", traceID).
			Msg("Request started")

		next.ServeHTTP(w, r)

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("service", "user-service").
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
