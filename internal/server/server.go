// Task 1.3.9: HTTP server initialization and lifecycle management
package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api"
)

// Config holds HTTP server configuration.
type Config struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultConfig returns default HTTP server configuration.
func DefaultConfig() Config {
	return Config{
		Host:         "0.0.0.0",
		Port:         8080,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Server wraps the HTTP server and database.
type Server struct {
	config Config
	db     *sql.DB
	http   *http.Server
}

// NewServer creates a new HTTP server with the given database and configuration.
// Task 1.3.9: Initialize HTTP server with database and routing
func NewServer(db *sql.DB, config Config) *Server {
	router := api.NewRouter(db)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	return &Server{
		config: config,
		db:     db,
		http:   httpServer,
	}
}

// Start starts the HTTP server and blocks until an error occurs.
func (s *Server) Start(ctx context.Context) error {
	fmt.Printf("Starting HTTP server on %s\n", s.http.Addr)
	return s.http.ListenAndServe()
}

// Shutdown gracefully shuts down the server and closes the database connection.
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("Shutting down server...")

	// Shutdown HTTP server
	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	// Close database connection
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("database close error: %w", err)
	}

	fmt.Println("Server shutdown complete")
	return nil
}
