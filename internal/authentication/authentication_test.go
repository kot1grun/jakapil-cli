package authentication

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const testEnvPrefix = "JAKAPIL_TEST_"

// fakePasswordKeystone serves a minimal Identity v3 password authentication
// (POST /v3/auth/tokens) and issues a fixed token. It returns credentials
// errors when the username or password does not match the expected values.
func fakePasswordKeystone(t *testing.T, username, password, project string) *httptest.Server {
	t.Helper()

	expires := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	body, err := json.Marshal(map[string]any{
		"token": map[string]any{
			"id":         "issued-token",
			"expires_at": expires.Format(time.RFC3339),
		},
	})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Auth struct {
				Identity struct {
					Methods  []string `json:"methods"`
					Password struct {
						User struct {
							Name     string `json:"name"`
							Password string `json:"password"`
						} `json:"user"`
					} `json:"password"`
				} `json:"identity"`
				Scope struct {
					Project struct {
						Name string `json:"name"`
					} `json:"project"`
				} `json:"scope"`
			} `json:"auth"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Auth.Identity.Password.User.Name != username ||
			req.Auth.Identity.Password.User.Password != password {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": {"code": 401, "title": "Unauthorized"}}`))
			return
		}
		if project != "" && req.Auth.Scope.Project.Name != project {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": {"code": 401, "title": "Unauthorized"}}`))
			return
		}
		w.Header().Set("X-Subject-Token", "issued-token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// setV3Env sets the openrc-style environment variables under the test prefix
// expected by clientconfig and restores the previous values on cleanup.
func setV3Env(t *testing.T, authURL, username, password, project string) {
	t.Helper()

	prev := map[string]string{}
	for _, k := range []string{"AUTH_URL", "USERNAME", "PASSWORD", "PROJECT_NAME"} {
		if old, ok := os.LookupEnv(testEnvPrefix + k); ok {
			prev[testEnvPrefix+k] = old
		}
	}

	unsetAll := map[string]string{
		testEnvPrefix + "AUTH_URL":            "",
		testEnvPrefix + "USERNAME":            "",
		testEnvPrefix + "PASSWORD":            "",
		testEnvPrefix + "PROJECT_NAME":        "",
		testEnvPrefix + "USER_DOMAIN_NAME":    "",
		testEnvPrefix + "DOMAIN_NAME":         "",
		testEnvPrefix + "PROJECT_DOMAIN_ID":   "",
		testEnvPrefix + "PROJECT_DOMAIN_NAME": "",
	}
	osSetenvAll(t, unsetAll)
	os.Setenv(testEnvPrefix+"AUTH_URL", authURL)
	os.Setenv(testEnvPrefix+"USERNAME", username)
	os.Setenv(testEnvPrefix+"PASSWORD", password)
	if project != "" {
		os.Setenv(testEnvPrefix+"PROJECT_NAME", project)
		os.Setenv(testEnvPrefix+"PROJECT_DOMAIN_NAME", "Default")
	}
	os.Setenv(testEnvPrefix+"DOMAIN_NAME", "Default")
	os.Setenv(testEnvPrefix+"USER_DOMAIN_NAME", "Default")

	t.Cleanup(func() {
		for full, old := range prev {
			_ = os.Setenv(full, old)
		}
	})
}

// osSetenvAll records the previous values of every key and registers a cleanup
// that restores them.
func osSetenvAll(t *testing.T, values map[string]string) {
	t.Helper()
	prev := map[string](*string){}
	for k := range values {
		if old, ok := os.LookupEnv(k); ok {
			cur := old
			prev[k] = &cur
		}
	}
	for k, v := range values {
		if v == "" {
			_ = os.Unsetenv(k)
		} else {
			_ = os.Setenv(k, v)
		}
	}
	t.Cleanup(func() {
		for k := range values {
			if p := prev[k]; p != nil {
				_ = os.Setenv(k, *p)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	})
}

func TestToken(t *testing.T) {
	const username = "demo-user"
	const password = "s3cr3t!"
	const project = "demo-project"

	srv := fakePasswordKeystone(t, username, password, project)
	setV3Env(t, srv.URL+"/v3", username, password, project)

	res, err := Token(context.Background(), Options{
		AuthURL:   srv.URL + "/v3",
		EnvPrefix: testEnvPrefix,
	})
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if res.Token != "issued-token" {
		t.Fatalf("token = %q, want %q", res.Token, "issued-token")
	}
	wantExp := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	if !res.ExpiresAt.Equal(wantExp) {
		t.Fatalf("expires_at = %v, want %v", res.ExpiresAt, wantExp)
	}
}

func TestTokenBadCredentials(t *testing.T) {
	const username = "demo-user"
	const password = "s3cr3t!"

	srv := fakePasswordKeystone(t, username, password, "demo-project")
	setV3Env(t, srv.URL+"/v3", username, "wrong-password", "demo-project")

	if _, err := Token(context.Background(), Options{
		AuthURL:   srv.URL + "/v3",
		EnvPrefix: testEnvPrefix,
	}); err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestTokenMissingCredentials(t *testing.T) {
	// A prefix with no configured environment variables must fail during
	// credential resolution, before any network I/O happens.
	_, err := Token(context.Background(), Options{
		AuthURL:   "http://127.0.0.1:1/v3",
		EnvPrefix: "DEFINITELY_UNSET_PREFIX_",
	})
	if err == nil {
		t.Fatal("expected error when no credentials are configured")
	}
}

func TestAuthURLFromFlag(t *testing.T) {
	const username = "u"
	const password = "p"
	project := ""
	setV3Env(t, "http://overridden.invalid/v3", username, password, project)

	// The authoritative endpoint comes from Options.AuthURL even though the
	// environment would point elsewhere; clientconfig reads the env, but
	// buildAuthOptions overrides the endpoint.
	srv := fakePasswordKeystone(t, username, password, "")
	res, err := Token(context.Background(), Options{
		AuthURL:   srv.URL + "/v3",
		EnvPrefix: testEnvPrefix,
	})
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if res.Token != "issued-token" {
		t.Fatalf("token = %q, want %q", res.Token, "issued-token")
	}
}
