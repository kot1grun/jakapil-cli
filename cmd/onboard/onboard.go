/*
Copyright © 2026 Kot1gruN
*/
package onboard

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
	"golang.org/x/term"

	"jakapil-cli/internal/authentication"
)

const (
	// DefaultAuthURL is the Selectel Keystone Identity v3 endpoint used when
	// the user leaves the auth URL prompt blank.
	DefaultAuthURL = "https://cloud.api.selcloud.ru/identity/v3"

	// accountIDDigits is the expected length of a Selectel account id, which is
	// used as the project (and user) domain name.
	accountIDDigits = 6

	// projectIDLength is the expected length of a Selectel project id: an
	// OpenStack UUID expressed as a 32 character hex string.
	projectIDLength = 32

	// authTimeout bounds the single authentication request issued to validate
	// the entered credentials.
	authTimeout = 30 * time.Second
)

// authConfig mirrors the "auth" block of a clouds.yaml profile. The field names
// follow the OpenStack client configuration schema.
type authConfig struct {
	AuthURL           string `yaml:"auth_url"`
	ProjectDomainName string `yaml:"project_domain_name"`
	UserDomainName    string `yaml:"user_domain_name"`
	ProjectID         string `yaml:"project_id"`
	Username          string `yaml:"username"`
}

// contextConfig is a single entry of the "contexts" map. Together with the
// auth block it holds the optional default region.
type contextConfig struct {
	Auth   *authConfig `yaml:"auth"`
	Region string      `yaml:"region,omitempty"`
}

// configFile is the on-disk structure of jakapil-cli.yaml. It mirrors
// clouds.yaml except that the top level object is called "contexts" instead of
// "clouds".
type configFile struct {
	Contexts map[string]*contextConfig `yaml:"contexts"`
}

// NewOnboardCommand returns the "onboard" command. It interactively collects
// the connection details for a Selectel project, validates them with a single
// authentication request (project scope), and writes the first named context
// "default" into the jakapil-cli.yaml configuration file. The password is used
// only to validate the connection and is never stored.
func NewOnboardCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Create the jakapil-cli.yaml configuration interactively",
		Long: `Create the jakapil-cli.yaml configuration interactively.

You will be asked for the Selectel connection details: auth URL, account id,
project id, service user name, and the service user password. The password is
used once to validate the credentials against the Keystone Identity service
(expects a 201 Created response) and is never saved to the configuration file.

On success the configuration is stored under:
    ` + defaultConfigPath() + `

The resulting file mirrors clouds.yaml, but the top level object is named
"contexts"; the first context is created under the name "default". Next, issue
an API token with: jakapil-cli authenticate token issue`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd)
		},
	}

	return cmd
}

// defaultConfigPath is the default location used in the command help text.
func defaultConfigPath() string {
	path, err := authentication.ConfigFilePath()
	if err != nil {
		return "<config-dir>/jakapil-cli.yaml"
	}
	return path
}

// run performs the interactive onboarding: it gathers the inputs, validates the
// credentials, and writes the configuration file. The password never leaves the
// process beyond the single validation request.
func run(cmd *cobra.Command) error {
	reader := bufio.NewReader(cmd.InOrStdin())
	errOut := cmd.ErrOrStderr()

	authURL, err := prompt(reader, errOut, "Enter the auth URL ["+DefaultAuthURL+"]: ", DefaultAuthURL)
	if err != nil {
		return err
	}

	accountID, err := prompt(reader, errOut, "Enter the account id (6 digits, e.g. 098765): ", "")
	if err != nil {
		return err
	}
	if !validAccountID(accountID) {
		return fmt.Errorf("invalid account id %q: it must be exactly %d digits", accountID, accountIDDigits)
	}

	projectID, err := prompt(reader, errOut, "Enter the project id (32 hex characters): ", "")
	if err != nil {
		return err
	}
	if !validProjectID(projectID) {
		return fmt.Errorf("invalid project id %q: it must be %d hex characters", projectID, projectIDLength)
	}

	username, err := prompt(reader, errOut, "Enter the service user name: ", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(username) == "" {
		return errors.New("service user name cannot be empty")
	}

	password, err := promptSecret(reader, cmd.InOrStdin(), errOut, "Enter the service user password (used once, never stored): ")
	if err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return errors.New("password cannot be empty")
	}

	region, err := prompt(reader, errOut, "Enter the default region [ru-3]: ", "ru-3")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), authTimeout)
	defer cancel()

	if err := authenticate(ctx, authURL, accountID, projectID, username, password); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	path, err := authentication.ConfigFilePath()
	if err != nil {
		return fmt.Errorf("failed to resolve the configuration file path: %w", err)
	}

	if err := saveConfig(path, authURL, accountID, projectID, username, region); err != nil {
		return fmt.Errorf("failed to save the configuration file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Authentication successful, configuration saved to '%s'.\n", path)
	fmt.Fprintln(cmd.OutOrStdout(), "Run \"jakapil-cli authenticate token issue\" to issue and manage your API token(s).")
	return nil
}

// prompt reads a single line from the buffered reader, printing prompt to w
// first. An empty input is replaced with def.
func prompt(r *bufio.Reader, w io.Writer, prompt, def string) (string, error) {
	fmt.Fprint(w, prompt)
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

// promptSecret reads the password without echoing it to the terminal when the
// input stream is an interactive terminal. For redirected or piped input
// (including tests) it falls back to reading a plain line from the reader.
func promptSecret(r *bufio.Reader, raw io.Reader, w io.Writer, prompt string) (string, error) {
	fmt.Fprint(w, prompt)

	if file, ok := raw.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		line, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(w)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(line)), nil
	}

	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// saveConfig serializes the first context "default" and writes it to path with
// owner-only permissions. The password is deliberately not part of the payload.
func saveConfig(path, authURL, accountID, projectID, username, region string) error {
	cfg := configFile{
		Contexts: map[string]*contextConfig{
			"default": {
				Auth: &authConfig{
					AuthURL:           authURL,
					ProjectDomainName: accountID,
					UserDomainName:    accountID,
					ProjectID:         projectID,
					Username:          username,
				},
				Region: region,
			},
		},
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// authenticate issues a project-scoped password token to the Keystone Identity
// service to validate the entered credentials, mirroring the documented request
// for an OpenStack project ("auth/tokens" with the password method). It requires
// the HTTP 201 Created response that a successful authentication yields.
func authenticate(ctx context.Context, endpoint, accountID, projectID, username, password string) error {
	payload := map[string]any{
		"auth": map[string]any{
			"identity": map[string]any{
				"methods": []string{"password"},
				"password": map[string]any{
					"user": map[string]any{
						"name":     username,
						"domain":   map[string]any{"name": accountID},
						"password": password,
					},
				},
			},
			"scope": map[string]any{
				"project": map[string]any{
					"id":     projectID,
					"domain": map[string]any{"name": accountID},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := strings.TrimRight(endpoint, "/") + "/auth/tokens"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		flat := strings.Join(strings.Fields(string(msg)), " ")
		if flat == "" {
			flat = "<empty response body>"
		}
		return fmt.Errorf("unexpected HTTP status %s: %s", resp.Status, flat)
	}

	return nil
}

func validAccountID(s string) bool {
	if len(s) != accountIDDigits {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func validProjectID(s string) bool {
	if len(s) != projectIDLength {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
