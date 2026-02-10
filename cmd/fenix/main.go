// FenixCRM - Agentic CRM OS
// Task 1.1: Project Setup - Entry point
// Following implementation plan exactly

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/matiasleandrokruk/fenix/internal/version"
)

func main() {
	// Parse command line flags
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Println(version.String())
		os.Exit(0)
	}

	// Handle help flag
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Default: print version (as per test requirement)
	fmt.Println(version.String())
}

func printHelp() {
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
	fmt.Println(helpText)
}
