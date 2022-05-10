package buildinfo

// These vars are set by -ldflags at build time.
var (
	Version   string
	GitCommit string
	BuildDate string
)
