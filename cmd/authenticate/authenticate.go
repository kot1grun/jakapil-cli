/*
Copyright © 2026 Kot1gruN
*/
package authenticate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"jakapil-cli/internal/authentication"
)

const (
	// FlagCloud is the flag that selects the clouds.yaml profile to use.
	FlagCloud = "cloud"

	// FlagConfigFile is the flag that points at a custom clouds.yaml file.
	FlagConfigFile = "config-file"

	// FlagEnvPrefix is the flag that overrides the environment variable prefix
	// (default "OS_") used to read openrc-style credentials.
	FlagEnvPrefix = "env-prefix"

	// FlagAuthURL is the flag that overrides the Keystone Identity v3 endpoint.
	FlagAuthURL = "auth-url"

	// FlagTokenFile is the flag that selects where the issued token is stored.
	FlagTokenFile = "token-file"

	// FlagDebug enables debug hints (such as the token expiry) on stderr.
	FlagDebug = "debug"

	// EnvCloud is the environment variable used as a fallback for FlagCloud.
	EnvCloud = "OS_CLOUD"

	// EnvConfigFile is the environment variable used as a fallback for FlagConfigFile.
	EnvConfigFile = "OS_CLIENT_CONFIG_FILE"

	// DefaultTokenFile is the default location where the acquired token is
	// persisted (unless overridden by --token-file).
	DefaultTokenFile = "~/.config/jakapil-cli/token"
)

// NewAuthenticateCommand returns the "authenticate" command. It logs in to the
// Selectel Keystone Identity service with credentials read from a clouds.yaml
// profile or the OS_* environment variables (as supplied by an openrc file),
// prints the issued API token and persists it to the token file.
func NewAuthenticateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authenticate",
		Short: "Authenticate to Selectel and obtain an API token",
		Long: `Authenticate to Selectel and obtain an API token.

Credentials are resolved from the environment variables exported by an openrc
file (OS_AUTH_URL, OS_USERNAME, OS_PASSWORD, ...) or from a clouds.yaml file
selected with --cloud. On success the issued token is printed and stored in the
token file (default ` + DefaultTokenFile + `), which is used by the other
commands as a token source in the absence of an explicit --token flag.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd)
		},
	}

	cmd.Flags().String(FlagCloud, "", fmt.Sprintf("clouds.yaml profile name to use (default $%s, or the only configured profile)", EnvCloud))
	cmd.Flags().String(FlagConfigFile, "", fmt.Sprintf("path to a clouds.yaml file (default $%s, or $HOME/.config/openstack/clouds.yaml)", EnvConfigFile))
	cmd.Flags().String(FlagEnvPrefix, "OS_", "environment variable prefix for openrc credentials")
	cmd.Flags().String(FlagAuthURL, "", "Selectel Keystone Identity v3 auth URL override (default $OS_AUTH_URL)")
	cmd.Flags().String(FlagTokenFile, DefaultTokenFile, "where to store the issued token")
	cmd.Flags().Bool(FlagDebug, false, "print the token expiry and storage path hints to stderr")

	return cmd
}

func run(cmd *cobra.Command) error {
	cloud, _ := cmd.Flags().GetString(FlagCloud)
	envPrefix, _ := cmd.Flags().GetString(FlagEnvPrefix)
	authURL, _ := cmd.Flags().GetString(FlagAuthURL)
	debug, _ := cmd.Flags().GetBool(FlagDebug)

	// Honor OS_CLIENT_CONFIG_FILE when --config-file is not set, so clientconfig
	// picks up the intended clouds.yaml. clientconfig already reads that env
	// var itself, but we document it here for clarity.
	if configFile, _ := cmd.Flags().GetString(FlagConfigFile); configFile != "" {
		_ = os.Setenv(EnvConfigFile, configFile)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := authentication.Token(ctx, authentication.Options{
		Cloud:     cloud,
		AuthURL:   authURL,
		EnvPrefix: envPrefix,
	})
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	tokenFile, _ := cmd.Flags().GetString(FlagTokenFile)
	tokenFile = authentication.ExpandHome(tokenFile)
	if err := authentication.SaveTokenRecord(tokenFile, authentication.StoredToken{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
	}); err != nil {
		return fmt.Errorf("failed to store token in %s: %w", tokenFile, err)
	}

	// Print only the token to stdout so the command stays convenient to pipe
	// into scripts and environment files. Debug hints go to stderr.
	fmt.Fprintln(cmd.OutOrStdout(), result.Token)

	if debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "token stored in %s\n", tokenFile)
		if !result.ExpiresAt.IsZero() {
			fmt.Fprintf(cmd.ErrOrStderr(), "token expires at %s\n", result.ExpiresAt.Format(time.RFC3339))
		}
	}

	return nil
}
