package version

var (
	// Version is the current version of the CLI. Set at build time via ldflags.
	Version = "dev"
	// Commit is the git commit hash at build time.
	Commit = "none"
	// Date is the build date.
	Date = "unknown"
	// BuiltBy is the entity that built the binary.
	BuiltBy = "unknown"
)
