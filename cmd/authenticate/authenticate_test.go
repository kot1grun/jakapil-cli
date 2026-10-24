package authenticate

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testEnvPrefix = "JAKAPIL_CLI_TEST_"

// fakePasswordKeystone serves a minimal Identity v3 password auth endpoint
// (POST /v3/auth/tokens) that validates username/password and returns a fixed
// token.
func fakePasswordKeystone(t *testing.T) *httptest.Server {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"token": map[string]any{
			"id":         "issued-token",
			"expires_at": "2099-01-01T00:00:00Z",
		},
	})
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("X-Subject-Token", "issued-token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// setCredentialsEnv sets openrc-style variables under the test prefix and
// restores previous values on cleanup.
func setCredentialsEnv(t *testing.T, authURL, username, password, project string) {
	t.Helper()

	prev := map[string]string{}
	for _, k := range []string{"AUTH_URL", "USERNAME", "PASSWORD", "PROJECT_NAME",
		"DOMAIN_NAME", "USER_DOMAIN_NAME", "PROJECT_DOMAIN_NAME"} {
		if old, ok := os.LookupEnv(testEnvPrefix + k); ok {
			prev[testEnvPrefix+k] = old
		}
	}

	os.Setenv(testEnvPrefix+"AUTH_URL", authURL)
	os.Setenv(testEnvPrefix+"USERNAME", username)
	os.Setenv(testEnvPrefix+"PASSWORD", password)
	os.Setenv(testEnvPrefix+"PROJECT_NAME", project)
	os.Setenv(testEnvPrefix+"DOMAIN_NAME", "Default")
	os.Setenv(testEnvPrefix+"USER_DOMAIN_NAME", "Default")
	os.Setenv(testEnvPrefix+"PROJECT_DOMAIN_NAME", "Default")

	t.Cleanup(func() {
		for k, old := range prev {
			_ = os.Setenv(k, old)
		}
		for _, k := range []string{"AUTH_URL", "USERNAME", "PASSWORD", "PROJECT_NAME",
			"DOMAIN_NAME", "USER_DOMAIN_NAME", "PROJECT_DOMAIN_NAME"} {
			if _, ok := prev[testEnvPrefix+k]; !ok {
				_ = os.Unsetenv(testEnvPrefix + k)
			}
		}
	})
}

func runCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewAuthenticateCommand()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestAuthenticateCommand(t *testing.T) {
	srv := fakePasswordKeystone(t)
	setCredentialsEnv(t, srv.URL+"/v3", "demo", "secret", "project")
	tokenFile := filepath.Join(t.TempDir(), "token")

	out, _, err := runCommand(t,
		"--env-prefix", testEnvPrefix,
		"--auth-url", srv.URL+"/v3",
		"--token-file", tokenFile,
	)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if got := strings.TrimSpace(out); got != "issued-token" {
		t.Fatalf("stdout = %q, want %q", got, "issued-token")
	}

	content, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("read token file: %v", err)
	}
	// The file now holds "token\nexpiry"; only the first line is the token.
	firstLine := strings.SplitN(strings.TrimSpace(string(content)), "\n", 2)[0]
	if firstLine != "issued-token" {
		t.Fatalf("token file = %q, want %q", firstLine, "issued-token")
	}
	if !strings.Contains(string(content), "2099-01-01T00:00:00Z") {
		t.Fatalf("token file missing expiry; got:\n%s", content)
	}
}

func TestAuthenticateCommandDebug(t *testing.T) {
	srv := fakePasswordKeystone(t)
	setCredentialsEnv(t, srv.URL+"/v3", "demo", "secret", "project")
	tokenFile := filepath.Join(t.TempDir(), "token")

	out, errOut, err := runCommand(t,
		"--env-prefix", testEnvPrefix,
		"--auth-url", srv.URL+"/v3",
		"--token-file", tokenFile,
		"--debug",
	)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if strings.TrimSpace(out) != "issued-token" {
		t.Fatalf("stdout = %q, want token only", out)
	}
	for _, want := range []string{"token stored in", "token expires at 2099-01-01T00:00:00Z"} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("stderr missing %q; got:\n%s", want, errOut)
		}
	}
}

func TestAuthenticateCommandFailsWithoutCredentials(t *testing.T) {
	_, _, err := runCommand(t, "--env-prefix", "DEFINITELY_UNSET_PREFIX_")
	if err == nil {
		t.Fatal("expected error without credentials")
	}
}
