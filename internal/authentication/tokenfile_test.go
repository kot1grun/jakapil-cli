package authentication

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestTokenFilePathDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test assumes a unix-style home layout")
	}

	// Force a known config dir so the result is deterministic even when the
	// ambient XDG_CONFIG_HOME differs.
	oldXDG, hadXDG := os.LookupEnv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", "/tmp/jakapil-config")
	t.Cleanup(func() {
		if hadXDG {
			os.Setenv("XDG_CONFIG_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	got, err := TokenFilePath("")
	if err != nil {
		t.Fatalf("TokenFilePath: %v", err)
	}
	if want := "/tmp/jakapil-config/jakapil-cli/token"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestTokenFilePathOverride(t *testing.T) {
	got, err := TokenFilePath("/custom/path/token")
	if err != nil {
		t.Fatalf("TokenFilePath: %v", err)
	}
	if want := "/custom/path/token"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "token")

	token := "some-secret-token"
	if err := SaveToken(path, token); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	got, err := LoadToken(path)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if got != token {
		t.Fatalf("token = %q, want %q", got, token)
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := fi.Mode().Perm(); perm != TokenFileMode {
			t.Fatalf("token file mode = %v, want %v", perm, TokenFileMode)
		}

		dirFi, err := os.Stat(filepath.Dir(path))
		if err != nil {
			t.Fatalf("stat dir: %v", err)
		}
		if perm := dirFi.Mode().Perm(); perm != 0o700 {
			t.Fatalf("token dir mode = %v, want 0700", perm)
		}
	}
}

func TestLoadTokenTrimsWhitespace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")
	if err := os.WriteFile(path, []byte("  abc-token\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := LoadToken(path)
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if got != "abc-token" {
		t.Fatalf("token = %q, want %q", got, "abc-token")
	}
}

func TestLoadTokenMissingFile(t *testing.T) {
	got, err := LoadToken(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("LoadToken: %v", err)
	}
	if got != "" {
		t.Fatalf("token = %q, want empty", got)
	}
}

func TestStoredTokenRoundTrip(t *testing.T) {
	// Redirect the default config dir to a temporary location.
	oldXDG, hadXDG := os.LookupEnv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() {
		if hadXDG {
			os.Setenv("XDG_CONFIG_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	path, err := TokenFilePath("")
	if err != nil {
		t.Fatalf("TokenFilePath: %v", err)
	}

	if err := SaveToken(path, "persisted-token"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	got, err := LoadStoredToken()
	if err != nil {
		t.Fatalf("LoadStoredToken: %v", err)
	}
	if got != "persisted-token" {
		t.Fatalf("token = %q, want %q", got, "persisted-token")
	}
}

func TestLoadStoredTokenMissingFile(t *testing.T) {
	oldXDG, hadXDG := os.LookupEnv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() {
		if hadXDG {
			os.Setenv("XDG_CONFIG_HOME", oldXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	got, err := LoadStoredToken()
	if err != nil {
		t.Fatalf("LoadStoredToken: %v", err)
	}
	if got != "" {
		t.Fatalf("token = %q, want empty", got)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}

	cases := map[string]struct {
		in   string
		want string
	}{
		"tilde slash":       {"~/x", filepath.Join(home, "x")},
		"bare tilde":        {"~", "~"},
		"absolute":          {"/x/y", "/x/y"},
		"relative":          {"x/y", "x/y"},
		"nothing to expand": {"", ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ExpandHome(tc.in); got != tc.want {
				t.Fatalf("ExpandHome(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSaveTokenRecordWithExpiry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	expiry := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	if err := SaveTokenRecord(path, StoredToken{Token: "abc-token", ExpiresAt: expiry}); err != nil {
		t.Fatalf("SaveTokenRecord: %v", err)
	}

	record, err := LoadTokenRecord(path)
	if err != nil {
		t.Fatalf("LoadTokenRecord: %v", err)
	}
	if record.Token != "abc-token" {
		t.Fatalf("token = %q, want %q", record.Token, "abc-token")
	}
	if !record.ExpiresAt.Equal(expiry) {
		t.Fatalf("ExpiresAt = %v, want %v", record.ExpiresAt, expiry)
	}
}

func TestLoadUsableStoredTokenBackwardCompat(t *testing.T) {
	// An existing one-line token file (written before expiry support) has a zero
	// ExpiresAt and must be treated as usable.
	dir := t.TempDir()
	old, had := os.LookupEnv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", dir)
	t.Cleanup(func() {
		if had {
			os.Setenv("XDG_CONFIG_HOME", old)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})
	path, err := TokenFilePath("")
	if err != nil {
		t.Fatalf("TokenFilePath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("legacy-token\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	token, err := LoadUsableStoredToken()
	if err != nil {
		t.Fatalf("LoadUsableStoredToken: %v", err)
	}
	if token != "legacy-token" {
		t.Fatalf("token = %q, want %q", token, "legacy-token")
	}
}

func TestLoadUsableStoredTokenExpired(t *testing.T) {
	dir := t.TempDir()
	old, had := os.LookupEnv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", dir)
	t.Cleanup(func() {
		if had {
			os.Setenv("XDG_CONFIG_HOME", old)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})
	path, err := TokenFilePath("")
	if err != nil {
		t.Fatalf("TokenFilePath: %v", err)
	}
	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := SaveTokenRecord(path, StoredToken{Token: "old-token", ExpiresAt: past}); err != nil {
		t.Fatalf("SaveTokenRecord: %v", err)
	}

	token, err := LoadUsableStoredToken()
	if token != "" {
		t.Fatalf("token = %q, want empty for expired token", token)
	}
	if err == nil {
		t.Fatal("expected error for expired token")
	}
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("error = %v, want ErrTokenExpired", err)
	}
}
