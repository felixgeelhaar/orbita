// Package api provides HTTP API handlers for the Orbita marketplace.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server is the HTTP API server for the marketplace.
type Server struct {
	mux     *http.ServeMux
	server  *http.Server
	logger  *slog.Logger
	handler *MarketplaceHandler
}

// ServerConfig holds configuration for the API server.
type ServerConfig struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultServerConfig returns the default server configuration.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Addr:         "0.0.0.0:8080",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// NewServer creates a new marketplace API server.
func NewServer(cfg ServerConfig, handler *MarketplaceHandler, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	mux := http.NewServeMux()

	s := &Server{
		mux:     mux,
		logger:  logger,
		handler: handler,
	}

	// Register routes
	s.registerRoutes()

	s.server = &http.Server{
		Addr:         cfg.Addr,
		Handler:      s.mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// registerRoutes sets up the API routes.
func (s *Server) registerRoutes() {
	// Health check
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// Marketplace API v1
	s.mux.HandleFunc("GET /api/v1/packages", s.handler.ListPackages)
	s.mux.HandleFunc("GET /api/v1/packages/search", s.handler.SearchPackages)
	s.mux.HandleFunc("GET /api/v1/packages/featured", s.handler.GetFeatured)
	s.mux.HandleFunc("GET /api/v1/packages/{packageID}", s.handler.GetPackage)
	s.mux.HandleFunc("GET /api/v1/packages/{packageID}/versions", s.handler.GetVersions)
	s.mux.HandleFunc("GET /api/v1/packages/{packageID}/versions/{version}", s.handler.GetVersion)
	s.mux.HandleFunc("GET /api/v1/packages/{packageID}/download", s.handler.DownloadPackage)

	// Publishers
	s.mux.HandleFunc("GET /api/v1/publishers", s.handler.ListPublishers)
	s.mux.HandleFunc("GET /api/v1/publishers/{slug}", s.handler.GetPublisher)
	s.mux.HandleFunc("GET /api/v1/publishers/{slug}/packages", s.handler.GetPublisherPackages)
}

// handleHealth handles health check requests.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// Start starts the API server.
func (s *Server) Start() error {
	s.logger.Info("starting marketplace API server",
		"addr", s.server.Addr,
	)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down marketplace API server")
	return s.server.Shutdown(ctx)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// Log error but can't do much at this point
			slog.Error("failed to encode JSON response", "error", err)
		}
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error":   http.StatusText(status),
		"message": message,
	})
}

// APIError represents an API error.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common API errors
var (
	ErrBadRequest = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "bad_request",
		Message: "Invalid request",
	}
	ErrNotFound = &APIError{
		Status:  http.StatusNotFound,
		Code:    "not_found",
		Message: "Resource not found",
	}
	ErrInternalServer = &APIError{
		Status:  http.StatusInternalServerError,
		Code:    "internal_error",
		Message: "Internal server error",
	}
)
