package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// fakeKeystone serves a minimal Identity v3 candidate token lookup (GET
// /v3/auth/tokens) that returns a service catalog with an MKS entry exposing
// endpoints in two regions. catalog must be the raw "catalog" array value.
func fakeKeystone(t *testing.T, catalog any) *httptest.Server {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"token": map[string]any{
			"id":         "issued-token",
			"expires_at": "2099-01-01T00:00:00Z",
			"catalog":    catalog,
		},
	})
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v3/auth/tokens" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("X-Subject-Token", "issued-token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func mksCatalog() []any {
	return []any{
		map[string]any{
			"type": mksServiceType,
			"name": "Managed Kubernetes",
			"id":   "mks-service",
			"endpoints": []any{
				map[string]any{"interface": "public", "region": "ru-1", "url": "https://mks.ru-1.examle.net/v1"},
				map[string]any{"interface": "public", "region": "ru-2", "url": "https://mks.ru-2.examle.net/v1"},
				map[string]any{"interface": "admin", "region": "ru-1", "url": "https://mks-admin.ru-1.examle.net/v1"},
			},
		},
	}
}

// TestDiscoverMKSEndpointWiring verifies that the MKS wrapper forwards the MKS
// service type and the requested region to the shared keystone discovery. Full
// discovery behavior (region selection, public-only interface, debug output) is
// covered by the internal/keystone package tests.
func TestDiscoverMKSEndpointWiring(t *testing.T) {
	srv := fakeKeystone(t, mksCatalog())
	identityEndpoint := srv.URL + "/v3"

	t.Run("resolves endpoint for requested region", func(t *testing.T) {
		got, err := discoverMKSEndpoint(identityEndpoint, "test-token", "ru-2", nil)
		if err != nil {
			t.Fatalf("discoverMKSEndpoint: %v", err)
		}
		if want := "https://mks.ru-2.examle.net/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("errors for unknown region", func(t *testing.T) {
		if _, err := discoverMKSEndpoint(identityEndpoint, "test-token", "nope", nil); err == nil {
			t.Fatal("expected error for unknown region")
		}
	})
}

// tokenValidatedKeystone serves the MKS catalog but returns 401 unless the
// X-Auth-Token header equals want. This lets tests assert which token source
// NewClient actually used.
func tokenValidatedKeystone(t *testing.T, want string) *httptest.Server {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"token": map[string]any{
			"id":         "issued-token",
			"expires_at": "2099-01-01T00:00:00Z",
			"catalog":    mksCatalog(),
		},
	})
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != want {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-Subject-Token", "issued-token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// mksCommand builds a cobra command with the flags NewClient expects.
func mksCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String(FlagToken, "", "")
	cmd.Flags().String(FlagEndpoint, DefaultEndpoint, "")
	cmd.Flags().String(FlagRegion, "", "")
	cmd.Flags().Bool(FlagDebug, false, "")
	return cmd
}

func TestNewClientUsesStoredTokenFile(t *testing.T) {
	const fileToken = "file-token"
	const envToken = "env-token"

	// The parser only accepts the file token, proving NewClient preferred the
	// stored token file over the environment.
	srv := tokenValidatedKeystone(t, fileToken)
	endpoint := srv.URL + "/v3"

	// Write the token file under a temporary config dir.
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
	tokenPath := filepath.Join(dir, "jakapil-cli", "token")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(tokenPath, []byte(fileToken), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	// Set the environment token to a different value.
	os.Setenv(EnvToken, envToken)
	t.Cleanup(func() { os.Unsetenv(EnvToken) })

	cmd := mksCommand()
	_ = cmd.Flags().Set(FlagEndpoint, endpoint)

	if _, err := NewClient(cmd); err != nil {
		t.Fatalf("NewClient: %v", err)
	}
}

func TestNewClientFlagWinsOverEnv(t *testing.T) {
	const flagToken = "flag-token"
	const envToken = "env-token"

	srv := tokenValidatedKeystone(t, flagToken)
	cmd := mksCommand()
	_ = cmd.Flags().Set(FlagEndpoint, srv.URL+"/v3")
	_ = cmd.Flags().Set(FlagToken, flagToken)

	os.Setenv(EnvToken, envToken)
	t.Cleanup(func() { os.Unsetenv(EnvToken) })

	// Point XDG at an empty dir so no real stored token interferes.
	old, had := os.LookupEnv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() {
		if had {
			os.Setenv("XDG_CONFIG_HOME", old)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	if _, err := NewClient(cmd); err != nil {
		t.Fatalf("NewClient: %v", err)
	}
}
func TestNewClientReadsRegionFromOS_REGION_NAME(t *testing.T) {
	const wantRegion = "ru-2"
	const wantEndpoint = "https://mks.ru-2.examle.net/v1"

	// The catalog lists ru-1 and ru-2; the endpoint resolved for the region
	// selected from the environment lets us assert which one was used.
	srv := tokenValidatedKeystone(t, "test-token")
	cmd := mksCommand()
	_ = cmd.Flags().Set(FlagEndpoint, srv.URL+"/v3")
	_ = cmd.Flags().Set(FlagToken, "test-token")

	os.Setenv(EnvRegion, wantRegion)
	t.Cleanup(func() { os.Unsetenv(EnvRegion) })

	// Point XDG at an empty dir so no real stored token interferes.
	old, had := os.LookupEnv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Cleanup(func() {
		if had {
			os.Setenv("XDG_CONFIG_HOME", old)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	cli, err := NewClient(cmd)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if cli.Endpoint != wantEndpoint {
		t.Fatalf("endpoint = %q, want %q (region from $%s not honored)", cli.Endpoint, wantEndpoint, EnvRegion)
	}
}
