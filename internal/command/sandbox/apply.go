package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/signadot/cli/internal/clio"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/poll"
	"github.com/signadot/cli/internal/print"
	"github.com/signadot/cli/internal/spinner"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newApply(sandbox *config.Sandbox) *cobra.Command {
	cfg := &config.SandboxApply{Sandbox: sandbox}

	cmd := &cobra.Command{
		Use:   "apply -f FILENAME var1=val1 var2=val2 ...",
		Short: "Create or update a sandbox with variable expansion",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apply(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}

	cfg.AddFlags(cmd)

	return cmd
}

func substMap(args []string) (map[string]string, error) {
	substMap := map[string]string{}
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("arg %q is not in <var>=<value> form", arg)
		}
		varName, val := parts[0], parts[1]
		if err := checkVar(varName); err != nil {
			return nil, fmt.Errorf("arg %q has invalid variable %q", arg, varName)
		}
		substMap[varName] = val
	}
	return substMap, nil
}

var varPat = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.-]*$`)

func checkVar(varName string) error {
	if !varPat.MatchString(varName) {
		return fmt.Errorf("invalid variable name %q, should match %s", varPat)
	}
	return nil
}

func apply(cfg *config.SandboxApply, out, log io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" {
		return errors.New("must specify sandbox request file with '-f' flag")
	}
	substMap, err := substMap(args)
	if err != nil {
		return err
	}

	sbt, err := clio.LoadYAML[any](cfg.Filename)
	if err != nil {
		return err
	}
	if err := substTemplate(sbt, substMap); err != nil {
		return err
	}
	req, err := unstructuredToSandbox(*sbt)
	if err != nil {
		return err
	}

	params := sandboxes.NewApplySandboxParams().
		WithOrgName(cfg.Org).WithSandboxName(req.Name).WithData(req)
	result, err := cfg.Client.Sandboxes.ApplySandbox(params, nil)
	if err != nil {
		return err
	}
	resp := result.Payload

	fmt.Fprintf(log, "Created sandbox %q (routing key: %s) in cluster %q.\n\n",
		req.Name, resp.RoutingKey, *req.Spec.Cluster)

	if cfg.Wait {
		// Wait for the sandbox to be ready.
		if err := waitForReady(cfg, log, resp.Name); err != nil {
			fmt.Fprintf(log, "\nThe sandbox was created, but it may not be ready yet. To check status, run:\n\n")
			fmt.Fprintf(log, "  signadot sandbox get %v\n\n", req.Name)
			return err
		}
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		// Print info on how to access the sandbox.
		sbURL := cfg.SandboxDashboardURL(resp.RoutingKey)
		fmt.Fprintf(out, "\nDashboard page: %v\n\n", sbURL)

		if len(resp.Endpoints) > 0 {
			if err := printEndpointTable(out, resp.Endpoints); err != nil {
				return err
			}
		}
		return nil
	case config.OutputFormatJSON:
		return print.RawJSON(out, resp)
	case config.OutputFormatYAML:
		return print.RawYAML(out, resp)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

func substTemplate(sbt *any, substMap map[string]string) error {
	vars := map[string]string{}
	err := substTemplateRec(sbt, substMap, vars)
	if err != nil {
		return err
	}
	notExpanded := []string{}
	for k := range vars {
		if _, ok := substMap[k]; !ok {
			notExpanded = append(notExpanded, k)
		}
	}
	if len(notExpanded) > 0 {
		return fmt.Errorf("unexpanded variables: %s", strings.Join(notExpanded, ", "))
	}
	return nil
}

func substTemplateRec(sbt *any, substMap, vars map[string]string) error {
	switch x := (*sbt).(type) {
	case map[string]any:
		for k, v := range x {
			if err := substTemplateRec(&v, substMap, vars); err != nil {
				return err
			}
			x[k] = v
		}

	case []any:
		for _, v := range x {
			if err := substTemplateRec(&v, substMap, vars); err != nil {
				return err
			}
		}
	case string:
		*sbt = substString(x, substMap, vars)
	default:
	}
	return nil
}

var varRefRx = regexp.MustCompile(`\$\{([a-zA-Z][a-zA-Z0-9_.-]*)\}`)

func substString(s string, substMap, vars map[string]string) string {
	matches := varRefRx.FindAllStringSubmatchIndex(s, -1)
	if matches == nil {
		return s
	}
	result := []string{}
	cur, start, end := 0, 0, 0
	for i := range matches {
		start, end = matches[i][2], matches[i][3]
		// store any skipped string
		if cur < start-2 {
			result = append(result, s[cur:start-2]) // ${
		}
		v := s[start:end]
		end++ // }
		cur = end
		vars[v] = ""
		repl, ok := substMap[v]
		if !ok {
			// unsubstituted variables are handled
			// in substTemplate to report all of them
			// no error is reported here.
			continue
		}
		result = append(result, repl)
	}
	if end < len(s) {
		result = append(result, s[end:])
	}
	return strings.Join(result, "")
}

func unstructuredToSandbox(un any) (*models.Sandbox, error) {
	if err := port2Int(&un); err != nil {
		return nil, err
	}
	d, err := json.Marshal(un)
	if err != nil {
		return nil, err
	}
	var sb models.Sandbox
	if err := json.Unmarshal(d, &sb); err != nil {
		return nil, err
	}
	return &sb, nil
}

func port2Int(un *any) error {
	fmt.Printf("port2Int %#v %T\n", *un, *un)
	switch x := (*un).(type) {
	case map[string]any:
		for k, v := range x {
			fmt.Printf("looking at key %q\n", k)
			if k != "port" {
				if err := port2Int(&v); err != nil {
					return err
				}
				x[k] = v
				continue
			}
			ps, ok := v.(string)
			if !ok {
				continue
			}
			p, err := strconv.ParseInt(ps, 10, 32)
			if err != nil {
				return fmt.Errorf("port is not int: %q", ps)
			}
			x[k] = p
		}
	case []any:
		for i := range x {
			if err := port2Int(&x[i]); err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

func waitForReady(cfg *config.SandboxApply, out io.Writer, sandboxName string) error {
	fmt.Fprintf(out, "Waiting (up to --wait-timeout=%v) for sandbox to be ready...\n", cfg.WaitTimeout)

	params := sandboxes.NewGetSandboxParams().WithOrgName(cfg.Org).WithSandboxName(sandboxName)

	spin := spinner.Start(out, "Sandbox status")
	defer spin.Stop()

	err := poll.Until(cfg.WaitTimeout, func() bool {
		result, err := cfg.Client.Sandboxes.GetSandbox(params, nil)
		if err != nil {
			// Keep retrying in case it's a transient error.
			spin.Messagef("error: %v", err)
			return false
		}
		status := result.Payload.Status
		if !status.Ready {
			spin.Messagef("Not Ready: %s", status.Message)
			return false
		}
		spin.StopMessagef("Ready: %s", status.Message)
		return true
	})
	if err != nil {
		spin.StopFail()
		return err
	}
	return nil
}
