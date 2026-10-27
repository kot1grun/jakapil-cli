/*
Copyright © 2026 Kot1gruN
*/
package authenticate

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

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

	// FlagAskPass, when set, prompts for the password on stdin when it is not
	// already present in the environment (OS_PASSWORD or the --env-prefix
	// equivalent), and stores it in that variable for the duration of the run.
	FlagAskPass = "ask-pass"

	// FlagProjectScope requests a project-scoped token. It is the default scope:
	// it is implied when neither --project-scope nor --acc-scope is given.
	FlagProjectScope = "project-scope"

	// FlagAccScope requests an account- (domain-) scoped token, which unlocks
	// account-level management APIs. Passing both --project-scope and --acc-scope
	// issues both tokens.
	FlagAccScope = "acc-scope"

	// EnvCloud is the environment variable used as a fallback for FlagCloud.
	EnvCloud = "OS_CLOUD"

	// EnvConfigFile is the environment variable used as a fallback for FlagConfigFile.
	EnvConfigFile = "OS_CLIENT_CONFIG_FILE"

	// DefaultTokenFile is the default location where the acquired token is
	// persisted (unless overridden by --token-file).
	DefaultTokenFile = "~/.config/jakapil-cli/token"

	// envPasswordSuffix is appended to the resolved env prefix to form the name
	// of the password variable read by gophercloud clientconfig (for the default
	// prefix this is OS_PASSWORD).
	envPasswordSuffix = "PASSWORD"
)

// NewAuthenticateCommand returns the "authenticate" command, the parent for the
// low-level token management sub-commands. Use "token issue" to obtain an API
// token from credentials read from a clouds.yaml profile or the OS_* environment
// variables (as supplied by an openrc file).
func NewAuthenticateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authenticate",
		Short: "Authenticate to Selectel and obtain an API token",
		Long: `Authenticate to Selectel and obtain an API token.

Credentials are resolved from the environment variables exported by an openrc
file (OS_AUTH_URL, OS_USERNAME, OS_PASSWORD, ...) or from a clouds.yaml file
selected with --cloud. Run "authenticate token issue" to issue and store the
token(s), printing them to stdout for use in scripts and openrc files.`,
	}
	cmd.AddCommand(newTokenCommand())
	return cmd
}

// newTokenCommand returns the "authenticate token" command, the parent for
// issuing and inspecting persisted tokens.
func newTokenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage API tokens",
		Long:  "Manage the API tokens issued by the Selectel Keystone Identity service.",
	}
	cmd.AddCommand(newIssueTokenCommand())
	return cmd
}

// newIssueTokenCommand returns the "authenticate token issue" command. It
// issues a project-scoped token by default; --acc-scope requests an
// account-scoped token and both flags together issue both. The token(s) are
// printed to stdout and stored in the token file (default ` + DefaultTokenFile + `).
func newIssueTokenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Issue an API token and store it",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIssue(cmd)
		},
	}

	cmd.Flags().Bool(FlagProjectScope, false, "request a project-scoped token (default when no scope flag is given)")
	cmd.Flags().Bool(FlagAccScope, false, "request an account-scoped token; with --project-scope, issue both")
	cmd.Flags().String(FlagCloud, "", fmt.Sprintf("clouds.yaml profile name to use (default $%s, or the only configured profile)", EnvCloud))
	cmd.Flags().String(FlagConfigFile, "", fmt.Sprintf("path to a clouds.yaml file (default $%s, or $HOME/.config/openstack/clouds.yaml)", EnvConfigFile))
	cmd.Flags().String(FlagEnvPrefix, "OS_", "environment variable prefix for openrc credentials")
	cmd.Flags().String(FlagAuthURL, "", "Selectel Keystone Identity v3 auth URL override (default $OS_AUTH_URL)")
	cmd.Flags().String(FlagTokenFile, DefaultTokenFile, "where to store the issued token(s)")
	cmd.Flags().Bool(FlagDebug, false, "print the token expiry and storage path hints to stderr")
	cmd.Flags().Bool(FlagAskPass, false, "prompt for the password on stdin when OS_PASSWORD (or the --env-prefix equivalent) is not set")

	return cmd
}

func runIssue(cmd *cobra.Command) error {
	cloud, _ := cmd.Flags().GetString(FlagCloud)
	envPrefix, _ := cmd.Flags().GetString(FlagEnvPrefix)
	authURL, _ := cmd.Flags().GetString(FlagAuthURL)
	debug, _ := cmd.Flags().GetBool(FlagDebug)
	askPass, _ := cmd.Flags().GetBool(FlagAskPass)
	projectScope, _ := cmd.Flags().GetBool(FlagProjectScope)
	accountScope, _ := cmd.Flags().GetBool(FlagAccScope)

	// Honor OS_CLIENT_CONFIG_FILE when --config-file is not set, so clientconfig
	// picks up the intended clouds.yaml. clientconfig already reads that env
	// var itself, but we document it here for clarity.
	if configFile, _ := cmd.Flags().GetString(FlagConfigFile); configFile != "" {
		_ = os.Setenv(EnvConfigFile, configFile)
	}

	if err := ensurePassword(cmd, envPrefix, askPass); err != nil {
		return err
	}

	// A project token is the default scope and the one consumed by the services
	// (MKS, CRaaS), so with no explicit scope flag we issue exactly that.
	if !projectScope && !accountScope {
		projectScope = true
	}

	scopes := make([]authentication.Scope, 0, 2)
	if projectScope {
		scopes = append(scopes, authentication.ScopeProject)
	}
	if accountScope {
		scopes = append(scopes, authentication.ScopeAccount)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	records := make([]authentication.StoredToken, 0, len(scopes))
	for _, scope := range scopes {
		result, err := authentication.Token(ctx, authentication.Options{
			Cloud:     cloud,
			AuthURL:   authURL,
			EnvPrefix: envPrefix,
			Scope:     scope,
		})
		if err != nil {
			return fmt.Errorf("authentication failed (scope %s): %w", scope, err)
		}
		records = append(records, authentication.StoredToken{
			Token:     result.Token,
			IssuedAt:  result.IssuedAt,
			ExpiresAt: result.ExpiresAt,
			Scope:     scope,
		})
	}

	tokenFile, _ := cmd.Flags().GetString(FlagTokenFile)
	tokenFile = authentication.ExpandHome(tokenFile)
	if err := authentication.SaveTokens(tokenFile, records); err != nil {
		return fmt.Errorf("failed to store token(s) in %s: %w", tokenFile, err)
	}

	// Print the project token (when issued) to stdout so the command stays
	// convenient to pipe into scripts and environment files; the account token
	// follows on its own line when both were issued. Debug hints go to stderr.
	for _, record := range records {
		if record.Scope != authentication.ScopeProject {
			continue
		}
		fmt.Fprintln(cmd.OutOrStdout(), record.Token)
	}
	if accountScope {
		for _, record := range records {
			if record.Scope == authentication.ScopeAccount {
				fmt.Fprintln(cmd.OutOrStdout(), record.Token)
			}
		}
	}

	if debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "token(s) stored in %s\n", tokenFile)
		for _, record := range records {
			if !record.ExpiresAt.IsZero() {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s token expires at %s\n",
					record.Scope, record.ExpiresAt.Format(time.RFC3339))
			}
		}
	}

	return nil
}

// ensurePassword makes sure the password is available for authentication when
// --ask-pass is set. It writes the password to the environment under the
// resolved prefix (OS_PASSWORD for the default "OS_" prefix) so clientconfig
// picks it up. When the variable is already set it is left untouched.
func ensurePassword(cmd *cobra.Command, envPrefix string, askPass bool) error {
	if !askPass {
		return nil
	}

	prefix := envPrefix
	if prefix == "" {
		prefix = "OS_"
	}
	envVar := prefix + envPasswordSuffix
	if os.Getenv(envVar) != "" {
		return nil
	}

	pwd, err := readSecret("Password: ", cmd.InOrStdin(), cmd.ErrOrStderr())
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	if strings.TrimSpace(pwd) == "" {
		return errors.New("password cannot be empty")
	}
	if err := os.Setenv(envVar, pwd); err != nil {
		return fmt.Errorf("failed to set %s: %w", envVar, err)
	}
	return nil
}

// readSecret reads a single line of input without echoing it back when r is an
// interactive terminal. For piped or redirected input (including tests) it falls
// back to reading a plain line so the command keeps working non-interactively.
func readSecret(prompt string, r io.Reader, w io.Writer) (string, error) {
	fmt.Fprint(w, prompt)

	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		raw, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(w)
		return string(raw), err
	}

	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
