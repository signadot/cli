package traffic

import (
	"path/filepath"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/trafficwatch"
	"github.com/signadot/cli/internal/utils/system"
)

func outDir(oFmt config.OutputFormat) (string, error) {
	signadotDir, err := system.GetSignadotDir()
	if err != nil {
		return "", err
	}
	dirSuffix := trafficwatch.FormatSuffix(oFmt)
	if dirSuffix != "" {
		dirSuffix = "-" + dirSuffix[1:]
	}
	relDir := trafficwatch.DefaultDirRelative + dirSuffix
	return filepath.Join(signadotDir, relDir), nil
}
