package keystone

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
// /v3/auth/tokens) that returns the given service catalog. catalog must be the
// raw "catalog" array value.
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

func TestDiscoverEndpoint(t *testing.T) {
	const mksType = "managed-kubernetes"

	baseCatalog := func() []any {
		return []any{
			map[string]any{
				"type": mksType,
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

	identityFor := func(t *testing.T, catalog any) string {
		t.Helper()
		return fakeKeystone(t, catalog).URL + "/v3"
	}

	t.Run("selects requested region", func(t *testing.T) {
		got, err := DiscoverEndpoint(Options{
			IdentityEndpoint: identityFor(t, baseCatalog()),
			Token:            "test-token",
			ServiceType:      mksType,
			Region:           "ru-2",
		})
		if err != nil {
			t.Fatalf("DiscoverEndpoint: %v", err)
		}
		if want := "https://mks.ru-2.examle.net/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("empty region returns first public endpoint", func(t *testing.T) {
		got, err := DiscoverEndpoint(Options{
			IdentityEndpoint: identityFor(t, baseCatalog()),
			Token:            "test-token",
			ServiceType:      mksType,
		})
		if err != nil {
			t.Fatalf("DiscoverEndpoint: %v", err)
		}
		if want := "https://mks.ru-1.examle.net/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("unknown region errors", func(t *testing.T) {
		_, err := DiscoverEndpoint(Options{
			IdentityEndpoint: identityFor(t, baseCatalog()),
			Token:            "test-token",
			ServiceType:      mksType,
			Region:           "nope",
		})
		if err == nil {
			t.Fatal("expected error for unknown region")
		}
	})

	t.Run("missing service type errors", func(t *testing.T) {
		_, err := DiscoverEndpoint(Options{
			IdentityEndpoint: identityFor(t, baseCatalog()),
			Token:            "test-token",
			ServiceType:      "does-not-exist",
		})
		if err == nil {
			t.Fatal("expected error for missing service type")
		}
	})

	t.Run("uses only public interface endpoints", func(t *testing.T) {
		catalog := []any{
			map[string]any{
				"type": mksType,
				"name": "Managed Kubernetes",
				"id":   "mks-service",
				"endpoints": []any{
					map[string]any{"interface": "admin", "region": "ru-1", "url": "https://mks-admin.ru-1.examle.net/v1"},
					map[string]any{"interface": "internal", "region": "ru-1", "url": "https://mks-int.ru-1.examle.net/v1"},
					map[string]any{"interface": "public", "region": "ru-1", "url": "https://mks.ru-1.examle.net/v1"},
				},
			},
		}
		got, err := DiscoverEndpoint(Options{
			IdentityEndpoint: identityFor(t, catalog),
			Token:            "test-token",
			ServiceType:      mksType,
			Region:           "ru-1",
		})
		if err != nil {
			t.Fatalf("DiscoverEndpoint: %v", err)
		}
		if want := "https://mks.ru-1.examle.net/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})

	t.Run("trims trailing slash for version-suffixed endpoints", func(t *testing.T) {
		catalog := []any{
			map[string]any{
				"type": "container-registry",
				"name": "Container Registry",
				"id":   "cr-v1-service",
				"endpoints": []any{
					// NormalizeURL-style trailing slash that would break v1/v2
					// suffix checks on the Selectel client constructors.
					map[string]any{"interface": "public", "region": "", "url": "https://cr.selcloud.ru/api/v1/"},
				},
			},
		}
		got, err := DiscoverEndpoint(Options{
			IdentityEndpoint: identityFor(t, catalog),
			Token:            "test-token",
			ServiceType:      "container-registry",
		})
		if err != nil {
			t.Fatalf("DiscoverEndpoint: %v", err)
		}
		if want := "https://cr.selcloud.ru/api/v1"; got != want {
			t.Fatalf("endpoint = %q, want %q", got, want)
		}
	})
}

func TestDiscoverRegions(t *testing.T) {
	const mksType = "managed-kubernetes"

	catalog := func() []any {
		return []any{
			map[string]any{
				"type": mksType,
				"name": "Managed Kubernetes",
				"id":   "mks-service",
				"endpoints": []any{
					map[string]any{"interface": "public", "region": "ru-2", "url": "https://mks.ru-2.examle.net/v1"},
					map[string]any{"interface": "public", "region": "ru-1", "url": "https://mks.ru-1.examle.net/v1"},
					// admin endpoint for an existing region must not be counted.
					map[string]any{"interface": "admin", "region": "ru-1", "url": "https://mks-admin.ru-1.examle.net/v1"},
					// non-regional endpoint (e.g. CRaaS-style) must be skipped.
					map[string]any{"interface": "public", "region": "", "url": "https://cr.selcloud.ru/api/v1"},
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

	t.Run("returns sorted public regions for the service type", func(t *testing.T) {
		srv := fakeKeystone(t, catalog())
		got, err := DiscoverRegions(Options{
			IdentityEndpoint: srv.URL + "/v3",
			Token:            "test-token",
			ServiceType:      mksType,
		})
		if err != nil {
			t.Fatalf("DiscoverRegions: %v", err)
		}
		want := []string{"ru-1", "ru-2"}
		if len(got) != len(want) {
			t.Fatalf("regions = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("regions = %v, want %v", got, want)
			}
		}
	})

	t.Run("errors when no public regional endpoint for the type", func(t *testing.T) {
		onlyAdmin := []any{
			map[string]any{
				"type": mksType,
				"name": "Managed Kubernetes",
				"id":   "mks-service",
				"endpoints": []any{
					map[string]any{"interface": "admin", "region": "ru-1", "url": "https://mks-admin.ru-1.examle.net/v1"},
				},
			},
		}
		srv := fakeKeystone(t, onlyAdmin)
		if _, err := DiscoverRegions(Options{
			IdentityEndpoint: srv.URL + "/v3",
			Token:            "test-token",
			ServiceType:      mksType,
		}); err == nil {
			t.Fatal("expected error when no public regional endpoint exists")
		}
	})

	t.Run("errors for missing service type", func(t *testing.T) {
		srv := fakeKeystone(t, catalog())
		if _, err := DiscoverRegions(Options{
			IdentityEndpoint: srv.URL + "/v3",
			Token:            "test-token",
			ServiceType:      "does-not-exist",
		}); err == nil {
			t.Fatal("expected error for missing service type")
		}
	})
}

func TestDiscoverEndpointDebugOutput(t *testing.T) {
	const mksType = "managed-kubernetes"
	srv := fakeKeystone(t, []any{
		map[string]any{
			"type": mksType,
			"name": "Managed Kubernetes",
			"id":   "mks-service",
			"endpoints": []any{
				map[string]any{"interface": "public", "region": "ru-1", "url": "https://mks.ru-1.examle.net/v1"},
			},
		},
	})

	var buf bytes.Buffer
	if _, err := DiscoverEndpoint(Options{
		IdentityEndpoint: srv.URL + "/v3",
		Token:            "test-token",
		ServiceType:      mksType,
		Region:           "ru-1",
		Debug:            &buf,
	}); err != nil {
		t.Fatalf("DiscoverEndpoint: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		`service catalog`,
		fmt.Sprintf(`"type": %q`, mksType),
		fmt.Sprintf("interface=%-8s region=%s url=%s", "public", "ru-1", "https://mks.ru-1.examle.net/v1"),
		fmt.Sprintf("selected endpoint (service=%q, region=%q): https://mks.ru-1.examle.net/v1", mksType, "ru-1"),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("debug output missing %q; got:\n%s", want, out)
		}
	}
}
