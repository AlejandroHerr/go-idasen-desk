package version

import (
	"os"
	"runtime"
)

var (
	Version     = "dev"         //nolint:gochecknoglobals // this is set by the build process
	Commit      = "none"        //nolint:gochecknoglobals // this is set by the build process
	BuildTime   = "unknown"     //nolint:gochecknoglobals // this is set by the build process
	Environment = "development" //nolint:gochecknoglobals // this is set by the build process
)

// GetVersion returns a formatted version string.
func GetVersion() string {
	return Version
}

// GetCommit returns the git commit hash.
func GetCommit() string {
	return Commit
}

// GetBuildTime returns the build timestamp.
func GetBuildTime() string {
	return BuildTime
}

func GetGoVersion() string {
	return runtime.Version()
}

// GetVersionInfo returns the full version information.
func GetVersionInfo() string {
	return Version + " (commit: " + Commit + ", built at: " + BuildTime + ", go version: " + GetGoVersion() + ")"
}

func GetEnvironment() string {
	// Allow runtime override via env var
	envFromVar := os.Getenv("APP_ENV")
	if envFromVar != "" {
		return envFromVar
	}

	return Environment
}

func IsDevelopment() bool {
	return GetEnvironment() == "development"
}

func IsProduction() bool {
	return GetEnvironment() == "production"
}
