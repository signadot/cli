package bug

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"

	"github.com/signadot/cli/internal/buildinfo"
	"github.com/signadot/cli/internal/config"
	"github.com/spf13/cobra"
)

func New(cfg *config.API) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "bug",
		Short: "Report a bug",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bug(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

var (
	newBugURL = "https://github.com/signadot/community/issues/new?assignees=&labels=bug&template=bug_report.md&title=%5Bbug%5D"
	pause     = 5 * time.Second
)

type BugConfig struct {
	*config.API
	BuildInfo string
	Error     error `json:"error,omitempty"`
}

func bug(cfg *config.API, out, log io.Writer, args []string) error {
	// report error on api config init instead of bailing out
	bugCfg := &BugConfig{
		API:       cfg,
		BuildInfo: buildinfo.String(),
	}
	err := cfg.InitAPIConfig()
	if err != nil {
		bugCfg.Error = err
	}
	d, e := json.MarshalIndent(bugCfg, "", "  ")
	if e != nil {
		// d will be the error message anyway...
		d = []byte(e.Error())
	}
	fmt.Fprintf(out, `Opening browser to %s in %s.  Please copy and paste the
following into the bug report:
%s
`, newBugURL, pause, d)
	time.Sleep(pause)

	var (
		cmd       *exec.Cmd
		browseErr error
	)
	cmd, browseErr = execOpen(newBugURL)
	if browseErr == nil {
		browseErr = cmd.Run()
	}
	if browseErr != nil {
		return fmt.Errorf(`could not open browser: %w
Please visit %s to report this bug.
`, browseErr, newBugURL)

	}
	return nil
}

func execOpen(url string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", url), nil
	case "darwin":
		return exec.Command("open", url), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url), nil
	default:
		return nil, fmt.Errorf("unsupported os: %s", runtime.GOOS)
	}
}
