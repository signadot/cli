package auth

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/fatih/color"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/spinner"
	"github.com/signadot/cli/internal/utils/system"
	sdkauth "github.com/signadot/go-sdk/client/auth"
	"github.com/signadot/go-sdk/client/orgs"
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
	loginCfg.AddFlags(cmd)

	return cmd
}

func runLogin(cfg *config.AuthLogin, out io.Writer) error {
	var err error
	if cfg.WithAPIKey != "" {
		err = apiKeyLogin(cfg, out)
	} else {
		err = bearerTokenLogin(cfg, out)
	}
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Fprintf(out, "%s Successfully logged in\n", green("✓"))
	return nil
}

func apiKeyLogin(cfg *config.AuthLogin, out io.Writer) error {
	// init the API client with the provided api key
	if err := cfg.InitAPIConfigWithApiKey(cfg.WithAPIKey); err != nil {
		return err
	}

	spin := spinner.Start(out, "Checking provided API key")
	defer spin.Stop()

	// resolve the org from the api key
	org, err := resolveOrg(cfg)
	if err != nil {
		spin.StopFail()
		return err
	}

	// store the auth info
	err = auth.StoreAuthInKeyring(&auth.Auth{
		APIKey:  cfg.WithAPIKey,
		OrgName: org.Name,
	})
	if err != nil {
		spin.StopFail()
		return fmt.Errorf("failed to store auth info: %w", err)
	}
	return nil
}

func bearerTokenLogin(cfg *config.AuthLogin, out io.Writer) error {
	// init an unauthirezed API client
	if err := cfg.InitUnauthAPIConfig(); err != nil {
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

	// init the API client with the provided bearer token
	if err := cfg.InitAPIConfigWithBearerToken(token.AccessToken); err != nil {
		return err
	}

	// resolve the org from the bearer token
	org, err := resolveOrg(cfg)
	if err != nil {
		return err
	}

	// store the auth info
	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	err = auth.StoreAuthInKeyring(&auth.Auth{
		BearerToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		OrgName:      org.Name,
		ExpiresAt:    &expiresAt,
	})
	if err != nil {
		return fmt.Errorf("failed to store auth info: %w", err)
	}
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
	if err := system.OpenBrowser(code.VerificationURI); err != nil {
		// If browser opening fails, fall back to just showing the URL
		fmt.Fprintf(out, "Please visit: %s\n", code.VerificationURI)
	} else {
		fmt.Fprintf(out, "Opening browser at: %s\n", code.VerificationURI)
	}
	fmt.Fprintf(out, "to authenticate using the code: %s\n\n", code.UserCode)
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
			spin.StopFail()
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

func resolveOrg(cfg *config.AuthLogin) (*models.OrgsOrg, error) {
	res, err := cfg.Client.Orgs.GetOrgName(&orgs.GetOrgNameParams{}, nil)
	if err != nil {
		return nil, err
	}
	orgInfo := res.Payload
	if len(orgInfo.Orgs) == 0 {
		return nil, errors.New("could not resolve orgs")
	}
	return orgInfo.Orgs[0], nil
}
