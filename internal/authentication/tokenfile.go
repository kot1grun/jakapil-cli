package authentication

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// TokenFileName is the name of the file, inside the configuration directory,
// that holds the persisted API token.
const TokenFileName = "token"

// ConfigFileName is the name of the main configuration file, which lives
// next to the token file inside the configuration directory.
const ConfigFileName = "jakapil-cli.yaml"

// TokenFileMode is the permission bits applied to the token file. The token is
// a secret, so only the owner may read or write it.
const TokenFileMode os.FileMode = 0o600

// ConfigDir returns the directory that holds all jakapil-cli configuration
// files: $XDG_CONFIG_HOME/jakapil-cli, or ~/.config/jakapil-cli when
// XDG_CONFIG_HOME is not set. The token file and the main configuration file
// (ConfigFileName) are stored together in this directory. The XDG layout is
// used consistently (also on macOS, where os.UserConfigDir would otherwise
// point into ~/Library/Application Support) so the location matches the value
// printed by the "authenticate token issue" command.
func ConfigDir() (string, error) {
	base, ok := os.LookupEnv("XDG_CONFIG_HOME")
	if !ok || base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}

	return filepath.Join(base, "jakapil-cli"), nil
}

// ConfigFilePath returns the default path of the main configuration file under
// the user's configuration directory: ConfigDir()/jakapil-cli.yaml.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}

// TokenFilePath returns the path of the token file under the user's
// configuration directory: ConfigDir()/token. When a non-empty override is
// given it wins over the default location.
func TokenFilePath(override string) (string, error) {
	if override != "" {
		return override, nil
	}

	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, TokenFileName), nil
}

// StoredToken is a persisted API token together with the moments it was issued
// (IssuedAt) and expires (ExpiresAt), and the scope it was issued for
// (ScopeProject or ScopeAccount). A zero timestamp means the information is not
// recorded, for example when the record was written by an earlier CLI version.
type StoredToken struct {
	Token     string    `yaml:"token,omitempty"`
	IssuedAt  time.Time `yaml:"issued_at,omitempty"`
	ExpiresAt time.Time `yaml:"expires_at,omitempty"`
	Scope     Scope     `yaml:"scope,omitempty"`
}

// Scope is the token scope discriminator stored with each persisted token.
type Scope string

// The scopes a token can be issued for. The Selectel Keystone Identity service
// can issue project-scoped (iam_token_project_scoped) and account-scoped
// (iam_token_account_scoped) tokens; the account scope corresponds to the
// domain/account the user belongs to.
const (
	// ScopeProject marks a token scoped to a project. Services such as MKS and
	// CRaaS require this scope, so it is the default when issuing tokens.
	ScopeProject Scope = "project"

	// ScopeAccount marks a token scoped to the whole account.
	ScopeAccount Scope = "account"
)

// ErrTokenExpired is returned when the persisted token can no longer be used
// because its expiry time has passed.
var ErrTokenExpired = errors.New("stored token has expired")

// SaveTokens persists the given tokens to path as a YAML list. Each entry holds
// the token itself plus its issue/expiry timestamps and scope. The file is
// written atomically with owner-only permissions.
func SaveTokens(path string, records []StoredToken) error {
	content, err := yaml.Marshal(records)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, TokenFileMode); err != nil {
		return err
	}
	if err := os.Chmod(tmp, TokenFileMode); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	// Ensure the final file also has restricted permissions (Rename preserves
	// the source mode, but be defensive against pre-existing files).
	return os.Chmod(path, TokenFileMode)
}

// SaveTokenRecord persists a single token. An empty scope defaults to
// ScopeProject, the scope consumed by the services (MKS, CRaaS).
func SaveTokenRecord(path string, record StoredToken) error {
	if record.Scope == "" {
		record.Scope = ScopeProject
	}
	return SaveTokens(path, []StoredToken{record})
}

// SaveToken writes the token to path, without issue or expiry information. It
// is kept for callers that persist only the token itself.
func SaveToken(path, token string) error {
	return SaveTokenRecord(path, StoredToken{Token: token})
}

// LoadTokens reads the persisted YAML list of tokens from path. A missing or
// empty file yields an empty list without an error.
//
// Files written by earlier versions of the CLI held a single token as plain
// text (the token on the first line and an optional RFC3339 expiry on the
// second). Such files are still accepted and are treated as a project token
// without a recorded issue time.
func LoadTokens(path string) ([]StoredToken, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	if strings.TrimSpace(string(content)) == "" {
		return nil, nil
	}

	var records []StoredToken
	if err := yaml.Unmarshal(content, &records); err == nil {
		return records, nil
	}

	// Legacy single-token format: the token on the first line, an optional
	// RFC3339 expiry on the second. Such a record is consumed as a project
	// token (the scope every service used before account tokens existed).
	lines := strings.SplitN(string(content), "\n", 2)
	record := StoredToken{Token: strings.TrimSpace(lines[0]), Scope: ScopeProject}
	if len(lines) > 1 {
		if expiry, perr := time.Parse(time.RFC3339, strings.TrimSpace(lines[1])); perr == nil {
			record.ExpiresAt = expiry
		}
	}
	return []StoredToken{record}, nil
}

// projectToken returns the token record scoped to the project, treating a
// record with an empty scope (such as one written by an earlier CLI version) as
// a project token. It reports false when no project token is stored.
func projectToken(records []StoredToken) (StoredToken, bool) {
	for _, r := range records {
		if r.Scope == ScopeProject || r.Scope == "" {
			return r, true
		}
	}
	return StoredToken{}, false
}

// accountToken returns the token record scoped to the account. Unlike
// projectToken it does not treat an empty scope as a match: an account-scoped
// token can only be issued explicitly (authenticate token issue --account-scope)
// and must not be confused with the project-scoped default.
func accountToken(records []StoredToken) (StoredToken, bool) {
	for _, r := range records {
		if r.Scope == ScopeAccount {
			return r, true
		}
	}
	return StoredToken{}, false
}

// LoadToken reads the project token from path. It returns an empty string
// without an error when the file does not exist or holds no project token.
func LoadToken(path string) (string, error) {
	record, err := LoadTokenRecord(path)
	if err != nil {
		return "", err
	}
	return record.Token, nil
}

// LoadTokenRecord reads the project token record from path. A missing or empty
// file yields an empty record without an error; an unknown or absent scope or
// expiry leaves the corresponding fields at their zero values. The project
// token is the one consumed by the services (MKS, CRaaS).
func LoadTokenRecord(path string) (StoredToken, error) {
	records, err := LoadTokens(path)
	if err != nil {
		return StoredToken{}, err
	}
	record, ok := projectToken(records)
	if !ok {
		return StoredToken{}, nil
	}
	return record, nil
}

// LoadStoredToken reads the project token from the default token file location.
// It returns an empty string without an error when the file does not exist.
func LoadStoredToken() (string, error) {
	path, err := TokenFilePath("")
	if err != nil {
		return "", err
	}
	return LoadToken(path)
}

// LoadUsableStoredToken returns the project token stored under the default
// location, provided it has not expired. A token persisted without expiry
// information is considered usable. It returns an empty string without an error
// when no project token is stored, and an error wrapping ErrTokenExpired when a
// stored project token has already expired. Callers that received the token
// from a live source (flag, environment) should not use this function: only the
// stored token is dated.
func LoadUsableStoredToken() (string, error) {
	path, err := TokenFilePath("")
	if err != nil {
		return "", err
	}
	records, err := LoadTokens(path)
	if err != nil {
		return "", err
	}
	record, ok := projectToken(records)
	if !ok || record.Token == "" {
		return "", nil
	}
	if !record.ExpiresAt.IsZero() && record.ExpiresAt.Before(time.Now()) {
		return "", fmt.Errorf("%w at %s: run \"jakapil-cli authenticate token issue\" to renew it",
			ErrTokenExpired, record.ExpiresAt.Format(time.RFC3339))
	}
	return record.Token, nil
}

// LoadUsableAccountToken returns the account-scoped token stored under the
// default location, provided it has not expired. It is the counterpart of
// LoadUsableStoredToken for account-level APIs (such as the Cloud Platform
// Projects / Resell API). Unlike the project variant it never falls back to an
// empty scope: only a token explicitly issued with --account-scope is used.
// It returns an empty string without an error when no account-scoped token is
// stored, and an error wrapping ErrTokenExpired when the stored account token
// has already expired.
func LoadUsableAccountToken() (string, error) {
	path, err := TokenFilePath("")
	if err != nil {
		return "", err
	}
	records, err := LoadTokens(path)
	if err != nil {
		return "", err
	}
	record, ok := accountToken(records)
	if !ok || record.Token == "" {
		return "", nil
	}
	if !record.ExpiresAt.IsZero() && record.ExpiresAt.Before(time.Now()) {
		return "", fmt.Errorf("%w at %s: run \"jakapil-cli authenticate token issue --account-scope\" to renew it",
			ErrTokenExpired, record.ExpiresAt.Format(time.RFC3339))
	}
	return record.Token, nil
}

// ExpandHome replaces a leading "~/" with the user's home directory, leaving
// the input unchanged when it does not start with "~/" or the home directory
// cannot be determined (e.g. on odd containers). It is used for flag values
// such as the default --token-file path so that "~" works regardless of the
// invoking shell.
func ExpandHome(path string) string {
	if len(path) < 2 || path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return home + path[1:]
}
