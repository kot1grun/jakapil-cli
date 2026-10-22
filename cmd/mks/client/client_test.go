package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
