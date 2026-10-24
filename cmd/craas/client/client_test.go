/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// fakeKeystone serves a minimal Identity v3 candidate token lookup (GET
// /v3/auth/tokens) that returns a service catalog with CRaaS entries. catalog
// must be the raw "catalog" array value.
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

func craasCatalog() []any {
	catalogEntry := func(serviceType, name, id string, endpoints []any) []any {
		return []any{map[string]any{
			"type":      serviceType,
			"name":      name,
			"id":        id,
			"endpoints": endpoints,
		}}
	}
	v1 := catalogEntry(
		craasServiceType, "Container Registry", "cr-v1-service",
		[]any{
			map[string]any{"interface": "public", "region": "", "url": "https://cr.selcloud.ru/api/v1"},
		},
	)
	v2 := catalogEntry(
		craasServiceTypeV2, "Container Registry V2", "cr-v2-service",
		[]any{
			map[string]any{"interface": "public", "region": "", "url": "https://cr.selcloud.ru/api/v2"},
		},
	)
	return append(v1, v2...)
}

// TestDiscoverCRaasEndpointWiring verifies that the CRaaS wrapper forwards the
// requested service type (V1 vs V2) to the shared keystone discovery with an
// empty region. Full discovery behavior (public-only interface, debug output)
// is covered by the internal/keystone package tests.
func TestDiscoverCRaasEndpointWiring(t *testing.T) {
	srv := fakeKeystone(t, craasCatalog())
	identityEndpoint := srv.URL + "/v3"

	t.Run("resolves v1 endpoint via container-registry service type", func(t *testing.T) {
		got, err := discoverCRaasEndpoint(identityEndpoint, "test-token", craasServiceType, nil)
		if err != nil {
			t.Fatalf("discoverCRaasEndpoint: %v", err)
		}
		if want := "https://cr.selcloud.ru/api/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("resolves v2 endpoint via container-registry-v2 service type", func(t *testing.T) {
		got, err := discoverCRaasEndpoint(identityEndpoint, "test-token", craasServiceTypeV2, nil)
		if err != nil {
			t.Fatalf("discoverCRaasEndpoint: %v", err)
		}
		if want := "https://cr.selcloud.ru/api/v2"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})
}

// credentialsCommand builds a cobra command that registers the token, auth-url
// and debug flags mirrored by resolveCredentials, so tests can exercise the
// resolution logic in isolation.
func credentialsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String(FlagToken, "", "")
	cmd.Flags().String(FlagEndpoint, DefaultEndpoint, "")
	cmd.Flags().Bool(FlagDebug, false, "")
	return cmd
}

func TestResolveCredentialsPriority(t *testing.T) {
	const tokenFlag = "--flag-token"
	const tokenEnv = "--env-token"
	const tokenFile = "--file-token"

	// Point the default token file location at a temp dir.
	writeStoredToken := func(t *testing.T, token string) string {
		t.Helper()
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
		if err := os.WriteFile(tokenPath, []byte(token), 0o600); err != nil {
			t.Fatalf("write token: %v", err)
		}
		return tokenPath
	}

	t.Run("flag wins over env and file", func(t *testing.T) {
		writeStoredToken(t, tokenFile)
		cmd := credentialsCommand()
		_ = cmd.Flags().Set(FlagToken, tokenFlag)
		os.Setenv(EnvToken, tokenEnv)
		t.Cleanup(func() { os.Unsetenv(EnvToken) })

		token, _, _, err := resolveCredentials(cmd)
		if err != nil {
			t.Fatalf("resolveCredentials: %v", err)
		}
		if token != tokenFlag {
			t.Fatalf("token = %q, want flag %q", token, tokenFlag)
		}
	})

	t.Run("stored file wins over env", func(t *testing.T) {
		writeStoredToken(t, tokenFile)
		cmd := credentialsCommand()
		os.Setenv(EnvToken, tokenEnv)
		t.Cleanup(func() { os.Unsetenv(EnvToken) })

		token, _, _, err := resolveCredentials(cmd)
		if err != nil {
			t.Fatalf("resolveCredentials: %v", err)
		}
		if token != tokenFile {
			t.Fatalf("token = %q, want file %q", token, tokenFile)
		}
	})

	t.Run("env is used when flag and file are absent", func(t *testing.T) {
		old, had := os.LookupEnv("XDG_CONFIG_HOME")
		os.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Cleanup(func() {
			if had {
				os.Setenv("XDG_CONFIG_HOME", old)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}
		})
		cmd := credentialsCommand()
		os.Setenv(EnvToken, tokenEnv)
		t.Cleanup(func() { os.Unsetenv(EnvToken) })

		token, _, _, err := resolveCredentials(cmd)
		if err != nil {
			t.Fatalf("resolveCredentials: %v", err)
		}
		if token != tokenEnv {
			t.Fatalf("token = %q, want env %q", token, tokenEnv)
		}
	})

	t.Run("errors with hint when no source provides a token", func(t *testing.T) {
		old, had := os.LookupEnv("XDG_CONFIG_HOME")
		os.Setenv("XDG_CONFIG_HOME", t.TempDir())
		t.Cleanup(func() {
			if had {
				os.Setenv("XDG_CONFIG_HOME", old)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}
		})
		cmd := credentialsCommand()
		os.Unsetenv(EnvToken)

		_, _, _, err := resolveCredentials(cmd)
		if err == nil {
			t.Fatal("expected error without a token source")
		}
		if !strings.Contains(err.Error(), "authenticate") {
			t.Fatalf("error should hint at the authenticate command; got: %v", err)
		}
	})
}
