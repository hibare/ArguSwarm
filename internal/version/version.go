// Package version provides versioning and metadata for the application.
package version

var (
	// CurrentVersion string
	// CurrentVersion is the version of the application.
	// It is set during the build process using the -X flag.
	// For example: go build -ldflags "-X 'github.com/hibare/ArguSwarm/internal/version.CurrentVersion=1.0.0'"
	// This allows for dynamic versioning based on the build environment.
	CurrentVersion = "0.0.0"
)
