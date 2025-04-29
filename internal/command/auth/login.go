package auth

import (
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/spinner"
	sdkauth "github.com/signadot/go-sdk/client/auth"
	"github.com/signadot/go-sdk/models"
	"github.com/spf13/cobra"
)

func newLogin(cfg *config.Auth) *cobra.Command {
	loginCfg := &config.AuthLogin{Auth: cfg}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Signadot",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(loginCfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func runLogin(cfg *config.AuthLogin, out io.Writer) error {
	if err := cfg.UnauthInitAPIConfig(); err != nil {
		return err
	}

	// get a device code
	code, err := getDeviceCode(cfg)
	if err != nil {
		return err
	}

	// wait for user authentication
	token, err := waitForUserAuth(cfg, out, code)
	if err != nil {
		return err
	}

	// store the access token and org name
	if err := auth.StoreToken(token.AccessToken); err != nil {
		return fmt.Errorf("failed to store access token: %w", err)
	}
	if err := auth.StoreOrg(token.OrgName); err != nil {
		return fmt.Errorf("failed to store org name: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintf(out, "%s Successfully logged in\n", green("✓"))
	return nil
}

func getDeviceCode(cfg *config.AuthLogin) (*models.AuthdevicesCode, error) {
	res, err := cfg.Client.Auth.AuthDeviceGetCode(&sdkauth.AuthDeviceGetCodeParams{})
	if err != nil {
		return nil, fmt.Errorf("couldn't get device code: %w", err)
	}
	return res.Payload, nil
}

func waitForUserAuth(cfg *config.AuthLogin, out io.Writer,
	code *models.AuthdevicesCode) (*models.AuthdevicesToken, error) {
	fmt.Fprintf(out, "To authenticate, visit: "+code.VerificationURI+"\n\n")
	spin := spinner.Start(out, "Waiting for authentication")
	defer spin.Stop()

	interval := time.Duration(code.Interval)
	param := &sdkauth.AuthDeviceGetTokenParams{
		Data: &models.AuthdevicesTokenInput{
			DeviceCode: code.DeviceCode,
		},
	}

	for {
		time.Sleep(time.Second * interval)

		res, err := cfg.Client.Auth.AuthDeviceGetToken(param)
		if err != nil {
			return nil, fmt.Errorf("couldn't get device token: %w", err)
		}
		token := res.Payload

		switch token.Status {
		case "completed":
			return token, nil
		case "slow_down":
			// lets increase our interval
			interval += 1
		}
	}
}
