/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
			map[string]any{"interface": "admin", "region": "", "url": "https://cr-admin.selcloud.ru/api/v1"},
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

func TestDiscoverCRaasEndpoint(t *testing.T) {
	srv := fakeKeystone(t, craasCatalog())
	identityEndpoint := srv.URL + "/v3"

	t.Run("discovers v1 endpoint", func(t *testing.T) {
		got, err := discoverCRaasEndpoint(identityEndpoint, "test-token", craasServiceType, nil)
		if err != nil {
			t.Fatalf("discoverCRaasEndpoint: %v", err)
		}
		if want := "https://cr.selcloud.ru/api/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("discovers v2 endpoint", func(t *testing.T) {
		got, err := discoverCRaasEndpoint(identityEndpoint, "test-token", craasServiceTypeV2, nil)
		if err != nil {
			t.Fatalf("discoverCRaasEndpoint: %v", err)
		}
		if want := "https://cr.selcloud.ru/api/v2"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("uses only public interface endpoints", func(t *testing.T) {
		// The admin endpoint is listed first in the same entry; discovery must
		// still pick the public one.
		srvPublic := fakeKeystone(t, []any{
			map[string]any{
				"type": craasServiceType,
				"name": "Container Registry",
				"id":   "cr-v1-service",
				"endpoints": []any{
					map[string]any{"interface": "admin", "region": "", "url": "https://cr-admin.selcloud.ru/api/v1"},
					map[string]any{"interface": "public", "region": "", "url": "https://cr.selcloud.ru/api/v1"},
				},
			},
		})
		got, err := discoverCRaasEndpoint(srvPublic.URL+"/v3", "test-token", craasServiceType, nil)
		if err != nil {
			t.Fatalf("discoverCRaasEndpoint: %v", err)
		}
		if want := "https://cr.selcloud.ru/api/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("missing service type errors", func(t *testing.T) {
		if _, err := discoverCRaasEndpoint(identityEndpoint, "test-token", "nonexistent", nil); err == nil {
			t.Fatal("expected error for missing service type")
		}
	})

	t.Run("does not require a region", func(t *testing.T) {
		// A region-bound catalog entry (non-empty region) still resolves because
		// CRaaS is not regional.
		srvRegioned := fakeKeystone(t, []any{
			map[string]any{
				"type": craasServiceType,
				"name": "Container Registry",
				"id":   "cr-v1-service",
				"endpoints": []any{
					map[string]any{"interface": "public", "region": "ru-1", "url": "https://cr.ru-1.selcloud.ru/api/v1"},
				},
			},
		})
		got, err := discoverCRaasEndpoint(srvRegioned.URL+"/v3", "test-token", craasServiceType, nil)
		if err != nil {
			t.Fatalf("discoverCRaasEndpoint: %v", err)
		}
		if want := "https://cr.ru-1.selcloud.ru/api/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})
}

func TestDiscoverCRaasEndpointDebugOutput(t *testing.T) {
	srv := fakeKeystone(t, craasCatalog())
	identityEndpoint := srv.URL + "/v3"

	var buf bytes.Buffer
	if _, err := discoverCRaasEndpoint(identityEndpoint, "test-token", craasServiceType, &buf); err != nil {
		t.Fatalf("discoverCRaasEndpoint: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		`service catalog`,
		fmt.Sprintf(`"type": %q`, craasServiceType),
		fmt.Sprintf("interface=%-8s region=%s url=%s", "public", "", "https://cr.selcloud.ru/api/v1"),
		fmt.Sprintf("interface=%-8s region=%s url=%s", "admin", "", "https://cr-admin.selcloud.ru/api/v1"),
		fmt.Sprintf("debug: selected CRaaS endpoint (service=%q): https://cr.selcloud.ru/api/v1", craasServiceType),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("debug output missing %q; got:\n%s", want, out)
		}
	}
}
