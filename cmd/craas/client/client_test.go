/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
