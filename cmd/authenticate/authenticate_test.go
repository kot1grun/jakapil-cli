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
	"time"

	"jakapil-cli/internal/authentication"
)

const testEnvPrefix = "JAKAPIL_CLI_TEST_"

// fakePasswordKeystone serves a minimal Identity v3 password auth endpoint
// (POST /v3/auth/tokens) that validates username/password and returns a fixed
// token. It inspects the requested scope: a domain-scoped request (account
// token) yields "account-token", any other request yields "project-token".
func fakePasswordKeystone(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}

		tokenID := "project-token"
		issuedAt := "2099-01-01T00:00:00Z"
		var req struct {
			Auth struct {
				Scope struct {
					Domain struct {
						Name string `json:"name"`
						ID   string `json:"id"`
					} `json:"domain"`
				} `json:"scope"`
			} `json:"auth"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if scope := req.Auth.Scope; scope.Domain.Name != "" || scope.Domain.ID != "" {
			tokenID = "account-token"
		}

		body, err := json.Marshal(map[string]any{
			"token": map[string]any{
				"id":         tokenID,
				"issued_at":  issuedAt,
				"expires_at": "2099-01-01T01:00:00Z",
			},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Subject-Token", tokenID)
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
	cmd.SetArgs(append([]string{"token", "issue"}, args...))
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

// mustRead returns the raw bytes of path, fatal on error. It is only for
// producing readable test failure output.
func mustRead(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "<read error: " + err.Error() + ">"
	}
	return string(b)
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
	if got := strings.TrimSpace(out); got != "project-token" {
		t.Fatalf("stdout = %q, want %q", got, "project-token")
	}

	records, err := authentication.LoadTokens(tokenFile)
	if err != nil {
		t.Fatalf("load token file: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("token file holds %d record(s), want 1; content:\n%s", len(records), mustRead(tokenFile))
	}
	rec := records[0]
	if rec.Token != "project-token" {
		t.Fatalf("token = %q, want %q", rec.Token, "project-token")
	}
	if rec.Scope != authentication.ScopeProject {
		t.Fatalf("scope = %q, want %q", rec.Scope, authentication.ScopeProject)
	}
	wantIssued := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	if !rec.IssuedAt.Equal(wantIssued) {
		t.Fatalf("issued_at = %v, want %v", rec.IssuedAt, wantIssued)
	}
	wantExpiry := time.Date(2099, 1, 1, 1, 0, 0, 0, time.UTC)
	if !rec.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expires_at = %v, want %v", rec.ExpiresAt, wantExpiry)
	}
}

func TestAuthenticateCommandBothScopes(t *testing.T) {
	srv := fakePasswordKeystone(t)
	setCredentialsEnv(t, srv.URL+"/v3", "demo", "secret", "project")
	tokenFile := filepath.Join(t.TempDir(), "token")

	out, _, err := runCommand(t,
		"--env-prefix", testEnvPrefix,
		"--auth-url", srv.URL+"/v3",
		"--token-file", tokenFile,
		"--project-scope",
		"--account-scope",
	)
	if err != nil {
		t.Fatalf("authenticate token issue: %v", err)
	}
	if got := strings.TrimSpace(out); got != "project-token\naccount-token" {
		t.Fatalf("stdout = %q, want project-token then account-token", got)
	}

	records, err := authentication.LoadTokens(tokenFile)
	if err != nil {
		t.Fatalf("load token file: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("token file holds %d record(s), want 2; content:\n%s", len(records), mustRead(tokenFile))
	}
	var project, account *authentication.StoredToken
	for i := range records {
		switch records[i].Scope {
		case authentication.ScopeProject:
			project = &records[i]
		case authentication.ScopeAccount:
			account = &records[i]
		}
	}
	if project == nil || project.Token != "project-token" {
		t.Fatalf("missing/wrong project token: %+v", project)
	}
	if account == nil || account.Token != "account-token" {
		t.Fatalf("missing/wrong account token: %+v", account)
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
	if strings.TrimSpace(out) != "project-token" {
		t.Fatalf("stdout = %q, want token only", out)
	}
	for _, want := range []string{"token(s) stored in", "project token expires at 2099-01-01T01:00:00Z"} {
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

// runCommandWithInput is like runCommand but feeds input to the command's stdin,
// which --ask-pass reads the password from.
func runCommandWithInput(t *testing.T, input string, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewAuthenticateCommand()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(input))
	cmd.SetArgs(append([]string{"token", "issue"}, args...))
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

// setCredentialsEnvWithoutPassword sets private openrc variables under the test
// prefix, leaving the password variable unset so --ask-pass has to fill it in.
func setCredentialsEnvWithoutPassword(t *testing.T, authURL, username, project string) {
	t.Helper()

	prev := map[string]string{}
	keys := []string{"AUTH_URL", "USERNAME", "PROJECT_NAME", "DOMAIN_NAME",
		"USER_DOMAIN_NAME", "PROJECT_DOMAIN_NAME", "PASSWORD"}
	for _, k := range keys {
		if old, ok := os.LookupEnv(testEnvPrefix + k); ok {
			prev[testEnvPrefix+k] = old
		}
	}

	os.Setenv(testEnvPrefix+"AUTH_URL", authURL)
	os.Setenv(testEnvPrefix+"USERNAME", username)
	os.Setenv(testEnvPrefix+"PROJECT_NAME", project)
	os.Setenv(testEnvPrefix+"DOMAIN_NAME", "Default")
	os.Setenv(testEnvPrefix+"USER_DOMAIN_NAME", "Default")
	os.Setenv(testEnvPrefix+"PROJECT_DOMAIN_NAME", "Default")
	_ = os.Unsetenv(testEnvPrefix + "PASSWORD")

	t.Cleanup(func() {
		for k, old := range prev {
			_ = os.Setenv(k, old)
		}
		for _, k := range keys {
			if _, ok := prev[testEnvPrefix+k]; !ok {
				_ = os.Unsetenv(testEnvPrefix + k)
			}
		}
	})
}

func TestAuthenticateCommandAskPassPromptsForPassword(t *testing.T) {
	srv := fakePasswordKeystone(t)
	setCredentialsEnvWithoutPassword(t, srv.URL+"/v3", "demo", "project")
	tokenFile := filepath.Join(t.TempDir(), "token")

	out, errOut, err := runCommandWithInput(t, "typed-password\n",
		"--env-prefix", testEnvPrefix,
		"--auth-url", srv.URL+"/v3",
		"--token-file", tokenFile,
		"--ask-pass",
	)
	if err != nil {
		t.Fatalf("authenticate --ask-pass: %v", err)
	}
	if got := strings.TrimSpace(out); got != "project-token" {
		t.Fatalf("stdout = %q, want %q", got, "project-token")
	}
	if !strings.Contains(errOut, "Password: ") {
		t.Fatalf("stderr missing password prompt; got:\n%s", errOut)
	}
	if got := os.Getenv(testEnvPrefix + "PASSWORD"); got != "typed-password" {
		t.Fatalf("env password = %q, want %q", got, "typed-password")
	}
}

func TestAuthenticateCommandAskPassKeepsExistingEnvPassword(t *testing.T) {
	srv := fakePasswordKeystone(t)
	setCredentialsEnvWithoutPassword(t, srv.URL+"/v3", "demo", "project")
	// Provide the password in the environment so --ask-pass must not prompt.
	os.Setenv(testEnvPrefix+"PASSWORD", "already-set")
	t.Cleanup(func() { os.Unsetenv(testEnvPrefix + "PASSWORD") })
	tokenFile := filepath.Join(t.TempDir(), "token")

	out, errOut, err := runCommandWithInput(t, "should-not-be-read\n",
		"--env-prefix", testEnvPrefix,
		"--auth-url", srv.URL+"/v3",
		"--token-file", tokenFile,
		"--ask-pass",
	)
	if err != nil {
		t.Fatalf("authenticate --ask-pass: %v", err)
	}
	if strings.TrimSpace(out) != "project-token" {
		t.Fatalf("stdout = %q, want %q", out, "project-token")
	}
	if strings.Contains(errOut, "Password: ") {
		t.Fatalf("--ask-pass prompted although the env password was already set; stderr:\n%s", errOut)
	}
	if got := os.Getenv(testEnvPrefix + "PASSWORD"); got != "already-set" {
		t.Fatalf("env password overwritten = %q, want %q", got, "already-set")
	}
}

// runListCommand executes the "authenticate token list" command with the given
// args and returns its stdout, stderr and error.
func runListCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := NewAuthenticateCommand()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(append([]string{"token", "list"}, args...))
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestAuthenticateTokenList(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	issued := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	expires := time.Date(2099, 1, 1, 1, 0, 0, 0, time.UTC)
	err := authentication.SaveTokens(tokenFile, []authentication.StoredToken{
		{Token: "project-token", Scope: authentication.ScopeProject, IssuedAt: issued, ExpiresAt: expires},
		{Token: "account-token", Scope: authentication.ScopeAccount, IssuedAt: issued.Add(time.Hour), ExpiresAt: expires.Add(time.Hour)},
	})
	if err != nil {
		t.Fatalf("seed token file: %v", err)
	}

	out, _, err := runListCommand(t, "--token-file", tokenFile)
	if err != nil {
		t.Fatalf("authenticate token list: %v", err)
	}

	for _, want := range []string{
		"Scope", "Issued At", "Expires At", "Status",
		"project", "account",
		issued.Format(time.RFC3339), expires.Format(time.RFC3339),
		"valid",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q; got:\n%s", want, out)
		}
	}

	// The list must render the unified bordered table, not a custom format.
	if !strings.Contains(out, "╔") || !strings.Contains(out, "║") {
		t.Fatalf("expected the unified bordered table, got:\n%s", out)
	}

	// The token values are secrets and must never be printed.
	for _, secret := range []string{"project-token", "account-token"} {
		if strings.Contains(out, secret) {
			t.Fatalf("output leaked secret %q; got:\n%s", secret, out)
		}
	}
}

// TestAuthenticateTokenListJSON proves the long-form json output is a safe,
// redacted projection: it carries the listing columns but never the token value.
func TestAuthenticateTokenListJSON(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	issued := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	expires := issued.Add(time.Hour)
	if err := authentication.SaveTokens(tokenFile, []authentication.StoredToken{
		{Token: "project-token", Scope: authentication.ScopeProject, IssuedAt: issued, ExpiresAt: expires},
	}); err != nil {
		t.Fatalf("seed token file: %v", err)
	}

	out, _, err := runListCommand(t, "--token-file", tokenFile, "-o", "json")
	if err != nil {
		t.Fatalf("authenticate token list -o json: %v", err)
	}
	for _, want := range []string{`"scope": "project"`, `"status": "valid"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("json output missing %q; got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "project-token") || strings.Contains(out, "escaped project-token") {
		t.Fatalf("json output must never leak the token secret; got:\n%s", out)
	}
}

func TestAuthenticateTokenListExpired(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	past := time.Now().Add(-time.Hour)
	err := authentication.SaveTokens(tokenFile, []authentication.StoredToken{
		{Token: "expired-token", Scope: authentication.ScopeProject, IssuedAt: past, ExpiresAt: past},
	})
	if err != nil {
		t.Fatalf("seed token file: %v", err)
	}

	out, _, err := runListCommand(t, "--token-file", tokenFile)
	if err != nil {
		t.Fatalf("authenticate token list: %v", err)
	}
	if !strings.Contains(out, "expired") {
		t.Fatalf("output missing expired status; got:\n%s", out)
	}
	if strings.Contains(out, "expired-token") {
		t.Fatalf("output leaked secret %q; got:\n%s", "expired-token", out)
	}
}

func TestAuthenticateTokenListEmpty(t *testing.T) {
	// Point at a path that does not exist: the listing must degrade gracefully
	// instead of failing.
	tokenFile := filepath.Join(t.TempDir(), "does-not-exist")

	out, _, err := runListCommand(t, "--token-file", tokenFile)
	if err != nil {
		t.Fatalf("authenticate token list on missing file: %v", err)
	}
	if !strings.Contains(out, "none") {
		t.Fatalf("output missing friendly none line; got:\n%s", out)
	}
}

func TestTokenStatusUnits(t *testing.T) {
	now := time.Date(2099, 1, 1, 0, 30, 0, 0, time.UTC)
	past := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	if got := tokenStatus(authentication.StoredToken{ExpiresAt: time.Time{}}, now); got != "no expiry info" {
		t.Fatalf("no-expiry token status = %q, want %q", got, "no expiry info")
	}
	if got := tokenStatus(authentication.StoredToken{ExpiresAt: past}, now); got != "expired" {
		t.Fatalf("expired token status = %q, want %q", got, "expired")
	}
	if got := tokenStatus(authentication.StoredToken{ExpiresAt: now}, now); got != "valid" {
		t.Fatalf("valid token status = %q, want %q", got, "valid")
	}

	if got := scopeLabel(authentication.ScopeProject); got != "project" {
		t.Fatalf("scopeLabel(project) = %q, want project", got)
	}
	if got := scopeLabel(authentication.ScopeAccount); got != "account" {
		t.Fatalf("scopeLabel(account) = %q, want account", got)
	}
	if got := scopeLabel(authentication.Scope("")); got != "(unknown)" {
		t.Fatalf("scopeLabel(empty) = %q, want (unknown)", got)
	}
	if got := scopeLabel(authentication.Scope("custom")); got != "custom" {
		t.Fatalf("scopeLabel(custom) = %q, want custom", got)
	}
}
