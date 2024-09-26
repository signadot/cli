package test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/jsonexact"
	"github.com/signadot/cli/internal/utils"
	"github.com/signadot/go-sdk/client/tests"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newApply(testConfig *config.Test) *cobra.Command {
	cfg := &config.TestApply{
		Test: testConfig,
	}
	cmd := &cobra.Command{
		Use:   "apply -f FILENAME [ --set var1=val1 --set var2=val2 ... ]",
		Short: "Create or update a test with variable expansion",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return apply(cfg, cmd.OutOrStdout(), cmd.ErrOrStderr(), args)
		},
	}
	cfg.AddFlags(cmd)
	return cmd
}

func apply(cfg *config.TestApply, wOut, wErr io.Writer, args []string) error {
	if err := cfg.InitAPIConfig(); err != nil {
		return err
	}
	if cfg.Filename == "" {
		return errors.New("must specify test resource file with '-f' flag")
	}

	// Load the sandbox spec
	t, err := loadTest(cfg.Filename, cfg.TemplateVals, false /*forDelete */)
	if err != nil {
		return err
	}
	// Send the request to the SaaS
	params := tests.NewApplyTestParams().
		WithOrgName(cfg.Org).WithTestName(t.Name).WithData(t.Spec)
	result, err := cfg.Client.Tests.ApplyTest(params, nil)
	if err != nil {
		return err
	}
	if !result.IsSuccess() {
		return errors.New(result.Error())
	}
	return printTest(cfg.OutputFormat, wOut, result.Payload)
}

func loadTest(file string, tplVals config.TemplateVals, forDelete bool) (*models.Test, error) {
	template, err := utils.LoadUnstructuredTemplate(file, tplVals, forDelete)
	if err != nil {
		return nil, err
	}
	return unstructuredToTest(template)
}

func unstructuredToTest(un any) (*models.Test, error) {
	name, spec, err := utils.UnstructuredToNameAndSpec(un)
	if err != nil {
		return nil, err
	}
	d, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	t := &models.Test{Name: name}
	if err := jsonexact.Unmarshal(d, &t.Spec); err != nil {
		return nil, fmt.Errorf("couldn't parse YAML sandbox definition - %s",
			strings.TrimPrefix(err.Error(), "json: "))
	}
	return t, nil
}
