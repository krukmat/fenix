// Task 1.3.9: HTTP server initialization and lifecycle management
package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/api"
	configpkg "github.com/matiasleandrokruk/fenix/internal/infra/config"
	"github.com/matiasleandrokruk/fenix/internal/infra/eventbus"
	"github.com/matiasleandrokruk/fenix/internal/infra/llm"
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
		WriteTimeout: 2 * time.Minute,
		IdleTimeout:  2 * time.Minute,
	}
}

// Server wraps the HTTP server and database.
type Server struct {
	config Config
	db     *sql.DB
	http   *http.Server
	cancel context.CancelFunc
	bgCtx  context.Context
	bgWG   sync.WaitGroup
}

// NewServer creates a new HTTP server with the given database and configuration.
// Task 1.3.9: Initialize HTTP server with database and routing
func NewServer(db *sql.DB, config Config) (*Server, error) {
	appCfg := configpkg.Load()
	chatProvider, err := llm.NewChatProvider(appCfg)
	if err != nil {
		return nil, fmt.Errorf("server: create chat provider: %w", err)
	}
	embedProvider, err := llm.NewEmbedProvider(appCfg)
	if err != nil {
		return nil, fmt.Errorf("server: create embed provider: %w", err)
	}

	bgCtx, cancel := context.WithCancel(context.Background())
	s := &Server{
		config: config,
		db:     db,
		bgCtx:  bgCtx,
		cancel: cancel,
	}
	sharedBus := eventbus.New()

	router, err := api.NewRouterWithRuntime(db, appCfg, api.RouterRuntime{
		Bus:               sharedBus,
		BackgroundContext: bgCtx,
		StartBackground:   s.startBackground,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("server: build router: %w", err)
	}
	s.startRelationshipRuntime(sharedBus, chatProvider, embedProvider)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  config.IdleTimeout,
	}

	s.http = httpServer
	return s, nil
}

// Start starts the HTTP server and blocks until an error occurs.
func (s *Server) Start(_ context.Context) error {
	fmt.Printf("Starting HTTP server on %s\n", s.http.Addr)
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server and closes the database connection.
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("Shutting down server...")

	// Shutdown HTTP server
	if err := s.http.Shutdown(ctx); err != nil {
		if s.cancel != nil {
			s.cancel()
		}
		return fmt.Errorf("server shutdown error: %w", err)
	}

	if s.cancel != nil {
		s.cancel()
	}
	if err := s.waitBackground(ctx); err != nil {
		return fmt.Errorf("background shutdown error: %w", err)
	}

	// Close database connection
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("database close error: %w", err)
	}

	fmt.Println("Server shutdown complete")
	return nil
}

func (s *Server) startBackground(fn func()) {
	s.bgWG.Add(1)
	go func() {
		defer s.bgWG.Done()
		fn()
	}()
}

func (s *Server) waitBackground(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.bgWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
