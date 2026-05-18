// FenixCRM - Agentic CRM OS
// Task 1.1: Project Setup - Entry point
// Following implementation plan exactly

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
	"github.com/matiasleandrokruk/fenix/internal/server"
	"github.com/matiasleandrokruk/fenix/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(args []string, out io.Writer) int {
	if len(args) > 0 && args[0] == "serve" {
		return runServe(args[1:], out)
	}

	fs := flag.NewFlagSet("fenix", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	showVersion := fs.Bool("version", false, "Show version information")
	showHelp := fs.Bool("help", false, "Show help")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintln(out, version.String()) //nolint:errcheck
		return 0
	}

	if *showHelp {
		printHelp(out)
		return 0
	}

	// Default: print version (as per test requirement)
	fmt.Fprintln(out, version.String()) //nolint:errcheck
	return 0
}

func resolveDefaultPort() int {
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			return p
		}
	}
	return 8080
}

func runServe(args []string, out io.Writer) int {
	port, parseErr := parseServeFlags(args)
	if parseErr != nil {
		return 2
	}

	db, err := openServeDB()
	if err != nil {
		fmt.Fprintf(out, "db init failed: %v\n", err) //nolint:errcheck
		return 1
	}

	cfg := server.DefaultConfig()
	cfg.Port = port
	srv, err := server.NewServer(db, cfg)
	if err != nil {
		fmt.Fprintf(out, "server: init failed: %v\n", err) //nolint:errcheck
		return 1
	}

	sigCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(sigCtx)
	}()

	select {
	case startErr := <-errCh:
		if startErr != nil {
			fmt.Fprintf(out, "server failed: %v\n", startErr) //nolint:errcheck
			_ = srv.Shutdown(context.Background())
			return 1
		}
	case <-sigCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
			fmt.Fprintf(out, "server shutdown failed: %v\n", shutdownErr) //nolint:errcheck
			return 1
		}
	}

	return 0
}

func parseServeFlags(args []string) (int, error) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	port := fs.Int("port", resolveDefaultPort(), "HTTP port")
	if err := fs.Parse(args); err != nil {
		return 0, err
	}
	return *port, nil
}

func openServeDB() (*sql.DB, error) {
	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = "./data/fenixcrm.db"
	}

	db, err := sqlite.NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	if migrateErr := sqlite.MigrateUp(db); migrateErr != nil {
		_ = db.Close()
		return nil, migrateErr
	}
	return db, nil
}

func printHelp(out io.Writer) {
	helpText := `FenixCRM - Agentic CRM OS

Usage:
  fenix [options]

Options:
  --version    Show version information
  --help       Show this help message

Commands:
  serve        Start the server (default)
  migrate      Run database migrations

Examples:
  fenix --version
  fenix serve --port 8080
  fenix migrate up`
	fmt.Fprintln(out, helpText) //nolint:errcheck
}
