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
	"jakapil-cli/internal/output"
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
	// it is implied when neither --project-scope nor --account-scope is given.
	FlagProjectScope = "project-scope"

	// FlagAccScope requests an account- (domain-) scoped token, which unlocks
	// account-level management APIs. Passing both --project-scope and --account-scope
	// issues both tokens.
	FlagAccScope = "account-scope"

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

	// The persistent --output flag matches the other command groups (vpc/mks/
	// craas): "authenticate token list" renders the unified tabular output by
	// default, with json/yaml available through the flag. The token "issue"
	// command prints raw token values for scripting and ignores it.
	cmd.PersistentFlags().StringP("output", "o", output.Table, "output format: table|dt|json|yaml")

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
	cmd.AddCommand(newListTokenCommand())
	return cmd
}

// newIssueTokenCommand returns the "authenticate token issue" command. It
// issues a project-scoped token by default; --account-scope requests an
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

// newListTokenCommand returns the "authenticate token list" command. It prints
// the tokens persisted by "authenticate token issue": their scope (project or
// account), when they were issued and expire, and whether they are still
// usable. The token values are masked because they are secrets.
func newListTokenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored API tokens",
		Long: `List the API tokens persisted by "authenticate token issue".

Each stored token is shown together with its scope (project or account), the
moment it was issued and expires, and whether it is still usable. The token
value itself is masked because it is a secret.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd)
		},
	}

	cmd.Flags().String(FlagTokenFile, DefaultTokenFile, "token file to inspect")

	return cmd
}

// tokenHeader are the column names of the unified tabular output used by
// "authenticate token list".
var tokenHeader = []string{"Scope", "Issued At", "Expires At", "Status"}

// tokenView is the json/yaml-safe projection of a stored token. It mirrors the
// tabular columns (scope, issue/expiry timestamps, usability) but never carries
// the token value itself: that is a secret, so it is not rendered in any format.
type tokenView struct {
	Scope     string `json:"scope"`
	IssuedAt  string `json:"issued_at"`
	ExpiresAt string `json:"expires_at"`
	Status    string `json:"status"`
}

// runList implements the "authenticate token list" command. A missing token
// file or a file with no tokens yields a friendly "none" line instead of an
// error, so listing works before any token has been issued. Otherwise the
// records are rendered through the unified output plumbing: a bordered table by
// default, or the redacted per-token projection for json/yaml.
func runList(cmd *cobra.Command) error {
	tokenFile, _ := cmd.Flags().GetString(FlagTokenFile)
	tokenFile = authentication.ExpandHome(tokenFile)

	records, err := authentication.LoadTokens(tokenFile)
	if err != nil {
		return fmt.Errorf("failed to read token file %s: %w", tokenFile, err)
	}

	if len(records) == 0 {
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Tokens in %s:\n\n", tokenFile)
		fmt.Fprintln(w, "  none: run \"jakapil-cli authenticate token issue\" to issue and store a token")
		return nil
	}

	now := time.Now()
	if output.IsTable(cmd) {
		rows := output.RowsOfValues(records, func(r authentication.StoredToken) []string {
			return tokenRow(r, now)
		})
		return output.PrintTable(cmd, tokenHeader, rows)
	}

	views := make([]tokenView, 0, len(records))
	for _, r := range records {
		views = append(views, tokenView{
			Scope:     scopeLabel(r.Scope),
			IssuedAt:  formatTokenTime(r.IssuedAt),
			ExpiresAt: formatTokenTime(r.ExpiresAt),
			Status:    tokenStatus(r, now),
		})
	}
	return output.Print(cmd, views)
}

// tokenRow builds one table row for a stored token. Token values are never
// shown: they are secrets.
func tokenRow(r authentication.StoredToken, now time.Time) []string {
	return []string{
		scopeLabel(r.Scope),
		formatTokenTime(r.IssuedAt),
		formatTokenTime(r.ExpiresAt),
		tokenStatus(r, now),
	}
}

// scopeLabel maps a stored token scope to the human-readable label used in the
// listing output. Unknown scopes are shown verbatim so the output stays honest.
func scopeLabel(scope authentication.Scope) string {
	switch scope {
	case authentication.ScopeProject:
		return "project"
	case authentication.ScopeAccount:
		return "account"
	default:
		if scope == "" {
			return "(unknown)"
		}
		return string(scope)
	}
}

// formatTokenTime renders a timestamp for the listing output, or "-" when the
// record carries no timestamp (for example a legacy single-token file).
func formatTokenTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

// tokenStatus reports whether a stored token is still usable. A token with no
// recorded expiry is treated as usable. No secret material is shown.
func tokenStatus(r authentication.StoredToken, now time.Time) string {
	if r.ExpiresAt.IsZero() {
		return "no expiry info"
	}
	if r.ExpiresAt.Before(now) {
		return "expired"
	}
	return "valid"
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
