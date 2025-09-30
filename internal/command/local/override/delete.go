package override

import (
	"fmt"
	"io"

	"github.com/signadot/cli/internal/builder"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/go-sdk/client/sandboxes"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newDelete(cfg *config.LocalOverride) *cobra.Command {
	deleteCfg := &config.LocalOverrideDelete{LocalOverride: cfg}

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a traffic override",
		Long: `Delete an existing traffic override by name and sandbox.
This will remove the redirect and altflow middleware from the sandbox.

Example:
  signadot local override delete my-override --sandbox=my-sandbox`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd.OutOrStdout(), deleteCfg, args[0])
		},
	}
	deleteCfg.AddFlags(cmd)

	return cmd
}

func runDelete(out io.Writer, cfg *config.LocalOverrideDelete, name string) error {
	if err := cfg.InitLocalConfig(); err != nil {
		return err
	}

	// Initialize API client
	if err := cfg.API.InitAPIConfig(); err != nil {
		return err
	}

	// Validate required parameters
	if cfg.Sandbox == "" {
		return fmt.Errorf("--sandbox is required")
	}

	printOverrideProgress(out, fmt.Sprintf("Removing override %s from sandbox %s", name, cfg.Sandbox))

	// Get the sandbox
	sandbox, err := getSandboxForDelete(cfg)
	if err != nil {
		return err
	}

	// Verify the override exists in the sandbox
	if !overrideExistsInSandbox(sandbox, name) {
		return fmt.Errorf("override %s not found in sandbox %s", name, cfg.Sandbox)
	}

	// Delete the override from the sandbox
	if err := deleteOverrideFromSandbox(cfg, sandbox, name); err != nil {
		return err
	}

	printOverrideStatus(out, fmt.Sprintf("Override %s deleted successfully from sandbox %s", name, cfg.Sandbox), true)

	return nil
}

func getSandboxForDelete(cfg *config.LocalOverrideDelete) (*models.Sandbox, error) {
	sandboxParams := sandboxes.NewGetSandboxParams().WithOrgName(cfg.Org).WithSandboxName(cfg.Sandbox)

	resp, err := cfg.Client.Sandboxes.
		GetSandbox(sandboxParams, nil)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

func overrideExistsInSandbox(sandbox *models.Sandbox, overrideName string) bool {
	if sandbox.Spec.Routing == nil || sandbox.Spec.Routing.Forwards == nil {
		return false
	}

	for _, forward := range sandbox.Spec.Routing.Forwards {
		if forward.Name == overrideName {
			return true
		}
	}
	return false
}

func deleteOverrideFromSandbox(cfg *config.LocalOverrideDelete, sandbox *models.Sandbox, overrideName string) error {
	// Use the sandbox builder to delete the override
	sbBuilder := builder.
		BuildSandbox(cfg.Sandbox, builder.WithData(*sandbox)).
		SetMachineID().
		DeleteOverrideMiddleware(overrideName)

	sb, err := sbBuilder.Build()
	if err != nil {
		return err
	}

	// Apply the updated sandbox
	sbParams := sandboxes.
		NewApplySandboxParams().
		WithOrgName(cfg.Org).
		WithSandboxName(cfg.Sandbox).
		WithData(&sb)

	_, err = cfg.Client.Sandboxes.ApplySandbox(sbParams, nil)
	return err
}
