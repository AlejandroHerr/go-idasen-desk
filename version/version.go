package version

var (
	Version   = "dev"     //nolint:gochecknoglobals // this is set by the build process
	Commit    = "none"    //nolint:gochecknoglobals // this is set by the build process
	BuildTime = "unknown" //nolint:gochecknoglobals // this is set by the build process
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

// GetVersionInfo returns the full version information.
func GetVersionInfo() string {
	return Version + " (commit: " + Commit + ", built at: " + BuildTime + ")"
}
