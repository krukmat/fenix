// Package version provides version information for the binary.
// Task 1.1: Project Setup
package version

import "fmt"

// Version is the current version of the application.
// This is set at build time using -ldflags.
var Version = "dev"

// BuildTime is when the binary was built.
// This is set at build time using -ldflags.
var BuildTime = "unknown"

// String returns the formatted version information.
func String() string {
	return fmt.Sprintf("fenix version %s (built %s)", Version, BuildTime)
}
