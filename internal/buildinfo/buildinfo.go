package buildinfo

import "fmt"

// These vars are set by -ldflags at build time.
var (
	Version   string
	GitCommit string
	BuildDate string
)

func String() string {
	return fmt.Sprintf("%s %s %s", Version, GitCommit, BuildDate)
}
