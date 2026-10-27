package onboard

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestValidAccountID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"098765", true},
		{"000000", true},
		{"12345", false},   // too short
		{"1234567", false}, // too long
		{"12345a", false},  // not a digit
		{"", false},
	}
	for _, c := range cases {
		if got := validAccountID(c.in); got != c.want {
			t.Errorf("validAccountID(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestValidProjectID(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{strings.Repeat("a1", 16), true},
		{strings.Repeat("0", 32), true},
		{strings.Repeat("F", 32), true},  // uppercase hex accepted
		{strings.Repeat("g", 32), false}, // not hex
		{strings.Repeat("a", 31), false}, // too short
		{strings.Repeat("a", 33), false}, // too long
	}
	for _, c := range cases {
		if got := validProjectID(c.in); got != c.want {
			t.Errorf("validProjectID(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestSaveConfigCreatesDir(t *testing.T) {
	parent := t.TempDir()
	// target lives in a nested directory that does not exist yet.
	path := filepath.Join(parent, "nested", "deeper", "jakapil-cli.yaml")

	if err := saveConfig(path, DefaultAuthURL, "098765", strings.Repeat("ab", 16), "svc", ""); err != nil {
		t.Fatalf("saveConfig() into a missing parent dir: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created in the nested directory: %v", err)
	}
}

func TestAuthenticateCreated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/identity/v3/auth/tokens" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q, want application/json", ct)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"token":{"id":"abc"}}`)
	}))
	defer srv.Close()

	err := authenticate(context.Background(), srv.URL+"/identity/v3", "098765", strings.Repeat("ab", 16), "svc", "secret")
	if err != nil {
		t.Fatalf("authenticate() error = %v", err)
	}
}

func TestAuthenticateNotCreated(t *testing.T) {
	payload := `{"error":{"code":401,"title":"Unauthorized"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, payload)
	}))
	defer srv.Close()

	err := authenticate(context.Background(), srv.URL, "098765", strings.Repeat("ab", 16), "svc", "wrong")
	if err == nil {
		t.Fatal("authenticate() expected error for 401, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q does not mention HTTP status 401", err)
	}
}

func TestSaveConfigNoPassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jakapil-cli.yaml")
	accountID := "098765"

	if err := saveConfig(path, DefaultAuthURL, accountID, strings.Repeat("ab", 16), "svc-user", "ru-1"); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	// The password must never be persisted.
	if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "password:") {
		t.Fatalf("config file must not contain the password:\n%s", raw)
	}
	// The password should be absent, not just obfuscated.
	if strings.Contains(string(raw), accountID+"\"\n    password") {
		t.Fatalf("config unexpectedly contains a password field:\n%s", raw)
	}

	var cfg configFile
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	def, ok := cfg.Contexts["default"]
	if !ok {
		t.Fatalf("contexts.default missing: %s", raw)
	}
	if def.Auth.AuthURL != DefaultAuthURL {
		t.Errorf("auth_url = %q, want %q", def.Auth.AuthURL, DefaultAuthURL)
	}
	if def.Auth.ProjectDomainName != accountID {
		t.Errorf("project_domain_name = %q, want %q", def.Auth.ProjectDomainName, accountID)
	}
	// user_domain_name is set implicitly and must equal project_domain_name.
	if def.Auth.UserDomainName != accountID {
		t.Errorf("user_domain_name = %q, want %q (equal to project_domain_name)", def.Auth.UserDomainName, accountID)
	}
	if def.Auth.Username != "svc-user" {
		t.Errorf("username = %q", def.Auth.Username)
	}
	if def.Region != "ru-1" {
		t.Errorf("region = %q, want ru-1", def.Region)
	}
}
