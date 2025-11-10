package config

import "github.com/spf13/cobra"

type Auth struct {
	*API
}

type AuthLogin struct {
	*Auth

	WithAPIKey string
	PlainText  bool
}

func (c *AuthLogin) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.WithAPIKey, "with-api-key", "", "log in using the provided API key.")
	cmd.Flags().BoolVar(&c.PlainText, "insecure-storage", false, "store credentials in plain text file instead of keyring")
}

type AuthStatus struct {
	*Auth
}

type AuthLogout struct {
	*Auth
}
