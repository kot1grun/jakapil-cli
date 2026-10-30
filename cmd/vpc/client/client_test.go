package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
)

// networkCatalog builds a service catalog with a networking ("network") entry
// exposing public endpoints in ru-1 and ru-2 (plus an admin endpoint that must
// be ignored). endpointFor maps region -> URL, defaulting to the given base.
func networkCatalog(ru1URL, ru2URL string) []any {
	return []any{
		map[string]any{
			"type": networkServiceType,
			"name": "Networking",
			"id":   "network-service",
			"endpoints": []any{
				map[string]any{"interface": "public", "region": "ru-1", "url": ru1URL},
				map[string]any{"interface": "public", "region": "ru-2", "url": ru2URL},
				map[string]any{"interface": "admin", "region": "ru-1", "url": ru1URL + "/admin"},
			},
		},
	}
}

// fakeIdentity serves an Identity v3 candidate token lookup (GET /v3/auth/tokens)
// that returns the given service catalog, plus a versions response for every
// other path so gophercloud's endpoint version probing succeeds.
func fakeIdentity(t *testing.T, catalog any) *httptest.Server {
	t.Helper()

	tokenBody, err := json.Marshal(map[string]any{
		"token": map[string]any{
			"id":         "issued-token",
			"expires_at": "2099-01-01T00:00:00Z",
			"catalog":    catalog,
		},
	})
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	versionsBody := []byte(`{"versions":[{"id":"v2.0","status":"CURRENT"}]}`)

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("X-Subject-Token", "issued-token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(tokenBody)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve gophercloud's version-probe GET (e.g. <endpoint>/versions).
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(versionsBody)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// networkCommand builds a cobra command with the flags NewNetworkServiceClients
// expects.
func networkCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String(FlagToken, "", "")
	cmd.Flags().String(FlagEndpoint, DefaultEndpoint, "")
	cmd.Flags().String(FlagRegion, "", "")
	cmd.Flags().Bool(FlagDebug, false, "")
	return cmd
}

// versionServer is a standalone networking endpoint that answers gophercloud's
// version-probe GET with a v2.0 version list.
func versionServer(t *testing.T) *httptest.Server {
	t.Helper()
	versionsBody := []byte(`{"versions":[{"id":"v2.0","status":"CURRENT"}]}`)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(versionsBody)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// TestNewNetworkServiceClientsAllRegions verifies that the all-regions builder
// discovers the public networking regions from the catalog once and produces a
// ServiceClient for each region, each pointing at that region's endpoint.
func TestNewNetworkServiceClientsAllRegions(t *testing.T) {
	// Two standalone networking endpoints so the per-region wiring is provable:
	// the networking clients must point at the URL of the region they were built
	// for, not at the identity server or each other.
	net1 := versionServer(t)
	net2 := versionServer(t)

	srv := fakeIdentity(t, networkCatalog(net1.URL, net2.URL))

	cmd := networkCommand()
	_ = cmd.Flags().Set(FlagEndpoint, srv.URL+"/v3")
	_ = cmd.Flags().Set(FlagToken, "test-token")

	regions, clients, err := NewNetworkServiceClients(cmd)
	if err != nil {
		t.Fatalf("NewNetworkServiceClients: %v", err)
	}

	wantRegions := []string{"ru-1", "ru-2"}
	if len(regions) != len(wantRegions) {
		t.Fatalf("regions = %v, want %v", regions, wantRegions)
	}
	for i, r := range wantRegions {
		if regions[i] != r {
			t.Fatalf("regions = %v, want %v", regions, wantRegions)
		}
	}

	wantEndpoint := map[string]string{
		"ru-1": net1.URL + "/",
		"ru-2": net2.URL + "/",
	}
	for _, r := range regions {
		cli, ok := clients[r]
		if !ok {
			t.Fatalf("no client for region %q", r)
		}
		if cli.Endpoint != wantEndpoint[r] {
			t.Fatalf("endpoint for %q = %q, want %q", r, cli.Endpoint, wantEndpoint[r])
		}
	}
}
