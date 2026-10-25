package authentication

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
// printed by the authenticate command.
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

// SaveToken writes the token to path, without expiry information. It is kept
// for callers that persist only the token itself.
func SaveToken(path, token string) error {
	return SaveTokenRecord(path, StoredToken{Token: token})
}

// StoredToken is the persisted API token together with the moment it expires.
// ExpiresAt is the zero time when the record carries no expiry information, for
// example when the token was written by an earlier version of the CLI.
type StoredToken struct {
	Token     string
	ExpiresAt time.Time
}

// ErrTokenExpired is returned when the persisted token can no longer be used
// because its expiry time has passed.
var ErrTokenExpired = errors.New("stored token has expired")

// SaveTokenRecord persists the token and, when ExpiresAt is not the zero time,
// its expiry timestamp (RFC3339) on a second line. Persisting the expiry lets
// clients detect stale tokens and ask the user to re-authenticate. The file is
// written atomically with owner-only permissions.
func SaveTokenRecord(path string, record StoredToken) error {
	content := record.Token
	if !record.ExpiresAt.IsZero() {
		content += "\n" + record.ExpiresAt.UTC().Format(time.RFC3339)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content+"\n"), TokenFileMode); err != nil {
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

// LoadToken reads the token (first line) from path. It returns an empty string
// without an error when the file does not exist or is empty.
func LoadToken(path string) (string, error) {
	record, err := LoadTokenRecord(path)
	if err != nil {
		return "", err
	}
	return record.Token, nil
}

// LoadTokenRecord reads the persisted token and its optional expiry. A missing
// or empty file yields an empty record without an error; an unknown or absent
// expiry leaves ExpiresAt at the zero time.
func LoadTokenRecord(path string) (StoredToken, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return StoredToken{}, nil
		}
		return StoredToken{}, err
	}

	lines := strings.SplitN(string(content), "\n", 2)
	record := StoredToken{Token: strings.TrimSpace(lines[0])}
	if len(lines) > 1 {
		if expiry, perr := time.Parse(time.RFC3339, strings.TrimSpace(lines[1])); perr == nil {
			record.ExpiresAt = expiry
		}
	}
	return record, nil
}

// LoadStoredToken reads the token from the default token file location. It
// returns an empty string without an error when the file does not exist.
func LoadStoredToken() (string, error) {
	path, err := TokenFilePath("")
	if err != nil {
		return "", err
	}
	return LoadToken(path)
}

// LoadUsableStoredToken returns the token stored under the default location,
// provided it has not expired. A token persisted without expiry information is
// considered usable. It returns an empty string without an error when no token
// is stored, and an error wrapping ErrTokenExpired when a stored token has
// already expired. Callers that received the token from a live source (flag,
// environment) should not use this function: only the stored token is dated.
func LoadUsableStoredToken() (string, error) {
	path, err := TokenFilePath("")
	if err != nil {
		return "", err
	}
	record, err := LoadTokenRecord(path)
	if err != nil {
		return "", err
	}
	if record.Token == "" {
		return "", nil
	}
	if !record.ExpiresAt.IsZero() && time.Now().After(record.ExpiresAt) {
		return "", fmt.Errorf("%w at %s: run \"jakapil-cli authenticate\" to renew it",
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
