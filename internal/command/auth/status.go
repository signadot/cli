package auth

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/docker/go-units"
	"github.com/signadot/cli/internal/auth"
	"github.com/signadot/cli/internal/config"
	"github.com/signadot/cli/internal/print"
	"github.com/spf13/cobra"
	"github.com/xeonx/timeago"
)

func newStatus(cfg *config.Auth) *cobra.Command {
	statusCfg := &config.AuthStatus{Auth: cfg}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(statusCfg, cmd.OutOrStdout())
		},
	}

	return cmd
}

func runStatus(cfg *config.AuthStatus, out io.Writer) error {
	authInfo, err := auth.ResolveAuth()
	if err != nil {
		return fmt.Errorf("could not resolve auth: %w", err)
	}

	switch cfg.OutputFormat {
	case config.OutputFormatDefault:
		return PrintAuthInfo(out, authInfo)
	case config.OutputFormatJSON:
		return printRawAuthInfo(out, print.RawJSON, authInfo)
	case config.OutputFormatYAML:
		return printRawAuthInfo(out, print.RawK8SYAML, authInfo)
	default:
		return fmt.Errorf("unsupported output format: %q", cfg.OutputFormat)
	}
}

func PrintAuthInfo(out io.Writer, authInfo *auth.ResolvedAuth) error {
	tw := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	// display status
	if authInfo == nil {
		fmt.Fprintln(tw, "Status:\tNot authenticated")
		return tw.Flush()
	}
	var status string
	if authInfo.ExpiresAt != nil && authInfo.ExpiresAt.Before(time.Now()) {
		status = "Not authenticated (expired token)"
	} else {
		status = "Authenticated"
		switch authInfo.Source {
		case auth.ConfigAuthSource:
			status += " (via config file or env vars)"
		case auth.PlainTextAuthSource:
			status += " (via plain text file)"
		}
	}
	fmt.Fprintf(tw, "Status:\t%s\n", status)

	// display token / api key
	if authInfo.BearerToken != "" {
		maskedToken := authInfo.BearerToken
		if len(maskedToken) > 32 {
			maskedToken = maskedToken[:32] + "..."
		}
		fmt.Fprintf(tw, "Token:\t%s\n", maskedToken)
	} else if authInfo.APIKey != "" {
		maskedAPIKey := authInfo.APIKey
		if len(maskedAPIKey) > 6 {
			maskedAPIKey = maskedAPIKey[:6] + "..."
		}
		fmt.Fprintf(tw, "API Key:\t%s\n", maskedAPIKey)
	}

	// display org
	fmt.Fprintf(tw, "Organization:\t%s\n", authInfo.OrgName)

	// display expiration
	if authInfo.ExpiresAt != nil {
		if authInfo.ExpiresAt.Before(time.Now()) {
			fmt.Fprintf(tw, "Expired:\t%s\n",
				timeago.NoMax(timeago.English).Format(*authInfo.ExpiresAt))
		} else {
			remaining := time.Until(*authInfo.ExpiresAt)
			fmt.Fprintf(tw, "Expires in:\t%s\n", units.HumanDuration(remaining))
		}
	}

	return tw.Flush()
}

func printRawAuthInfo(out io.Writer, printer func(out io.Writer, v any) error,
	authInfo *auth.ResolvedAuth) error {
	if authInfo != nil {
		if len(authInfo.BearerToken) > 32 {
			authInfo.BearerToken = authInfo.BearerToken[:32] + "..."
		}
		if len(authInfo.APIKey) > 6 {
			authInfo.APIKey = authInfo.APIKey[:6] + "..."
		}
	}
	return printer(out, authInfo)
}
