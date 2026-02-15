// FenixCRM - Agentic CRM OS
// Task 1.1: Project Setup - Entry point
// Following implementation plan exactly

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

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

func runServe(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	defaultPort := 8080
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			defaultPort = p
		}
	}
	port := fs.Int("port", defaultPort, "HTTP port")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = "./data/fenixcrm.db"
	}

	db, err := sqlite.NewDB(dbPath)
	if err != nil {
		fmt.Fprintf(out, "db init failed: %v\n", err) //nolint:errcheck
		return 1
	}
	if err := sqlite.MigrateUp(db); err != nil {
		fmt.Fprintf(out, "migrations failed: %v\n", err) //nolint:errcheck
		_ = db.Close()
		return 1
	}

	cfg := server.DefaultConfig()
	cfg.Port = *port
	srv := server.NewServer(db, cfg)

	if err := srv.Start(context.Background()); err != nil {
		fmt.Fprintf(out, "server failed: %v\n", err) //nolint:errcheck
		_ = srv.Shutdown(context.Background())
		return 1
	}

	return 0
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
