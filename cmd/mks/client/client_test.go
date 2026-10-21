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
// /v3/auth/tokens) that returns a service catalog with an "mks" entry exposing
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
		map[string]any{
			"type": "compute",
			"name": "Compute",
			"id":   "compute-service",
			"endpoints": []any{
				map[string]any{"interface": "public", "region": "ru-1", "url": "https://compute.ru-1.examle.net/v2.1"},
			},
		},
	}
}

func TestDiscoverMKSEndpointByRegion(t *testing.T) {
	catalog := mksCatalog()
	// The endpoint is used before the catalog: openstack.NewClient needs the
	// identity endpoint; we pass <server>/v3.
	srv := fakeKeystone(t, catalog)
	identityEndpoint := srv.URL + "/v3"

	t.Run("selects region ru-2", func(t *testing.T) {
		got, err := discoverMKSEndpoint(identityEndpoint, "test-token", "ru-2", nil)
		if err != nil {
			t.Fatalf("discoverMKSEndpoint: %v", err)
		}
		if want := "https://mks.ru-2.examle.net/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("empty region returns first public endpoint", func(t *testing.T) {
		got, err := discoverMKSEndpoint(identityEndpoint, "test-token", "", nil)
		if err != nil {
			t.Fatalf("discoverMKSEndpoint: %v", err)
		}
		if want := "https://mks.ru-1.examle.net/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("unknown region errors", func(t *testing.T) {
		if _, err := discoverMKSEndpoint(identityEndpoint, "test-token", "nope", nil); err == nil {
			t.Fatal("expected error for unknown region")
		}
	})

	t.Run("uses only public interface endpoints", func(t *testing.T) {
		// admin/internal endpoints are listed first; discovery must still pick
		// the public one.
		catalog := []any{
			map[string]any{
				"type": mksServiceType,
				"name": "Managed Kubernetes",
				"id":   "mks-service",
				"endpoints": []any{
					map[string]any{"interface": "admin", "region": "ru-1", "url": "https://mks-admin.ru-1.examle.net/v1"},
					map[string]any{"interface": "internal", "region": "ru-1", "url": "https://mks-int.ru-1.examle.net/v1"},
					map[string]any{"interface": "public", "region": "ru-1", "url": "https://mks.ru-1.examle.net/v1"},
				},
			},
		}
		srvPublic := fakeKeystone(t, catalog)
		got, err := discoverMKSEndpoint(srvPublic.URL+"/v3", "test-token", "ru-1", nil)
		if err != nil {
			t.Fatalf("discoverMKSEndpoint: %v", err)
		}
		if want := "https://mks.ru-1.examle.net/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})
}

func TestDiscoverMKSEndpointDebugOutput(t *testing.T) {
	srv := fakeKeystone(t, mksCatalog())
	identityEndpoint := srv.URL + "/v3"

	var buf bytes.Buffer
	if _, err := discoverMKSEndpoint(identityEndpoint, "test-token", "ru-1", &buf); err != nil {
		t.Fatalf("discoverMKSEndpoint: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		`service catalog`,
		fmt.Sprintf(`"type": %q`, mksServiceType),
		fmt.Sprintf("interface=%-8s region=%s url=%s", "public", "ru-1", "https://mks.ru-1.examle.net/v1"),
		fmt.Sprintf("interface=%-8s region=%s url=%s", "public", "ru-2", "https://mks.ru-2.examle.net/v1"),
		"selected MKS endpoint (region=\"ru-1\"): https://mks.ru-1.examle.net/v1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("debug output missing %q; got:\n%s", want, out)
		}
	}
}
