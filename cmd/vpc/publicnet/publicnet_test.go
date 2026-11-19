/*
Copyright © 2026 Kot1gruN
*/
package publicnet

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	publicnetapi "github.com/selectel/public-net-api-go/pkg/v1"
	"github.com/spf13/cobra"
)

const testToken = "test-token"

// testCommand builds a bare command with every public-net flag registered and
// a captured stdout buffer.
func testCommand(t *testing.T) (*cobra.Command, *strings.Builder) {
	t.Helper()
	cmd := &cobra.Command{Use: "public-net"}

	fs := cmd.Flags()
	fs.String("token", "", "")
	fs.String("auth-url", "", "")
	fs.String("region", "", "")
	fs.Bool("debug", false, "")
	fs.StringP("output", "o", "yaml", "output format: json or yaml")

	// list flags
	fs.String("project-id", "", "")
	fs.String("network-id", "", "")
	fs.String("description", "", "")
	fs.Bool("all-regions", false, "")

	// create/update flags
	fs.StringSlice("security-group-id", nil, "")
	fs.Bool("admin-state-up", false, "")

	// delete flags
	fs.Bool("force", false, "")

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&strings.Builder{})
	cmd.SetContext(context.Background())
	return cmd, &out
}

// newTestClient builds a Public Net API v1 client pinned to the given URL and
// the shared test token.
func newTestClient(server *httptest.Server) *publicnetapi.PublicNetAPIClient {
	c, err := publicnetapi.NewPublicNetAPIClient(&publicnetapi.Config{
		URL:       server.URL,
		AuthToken: testToken,
	})
	if err != nil {
		panic(err)
	}
	return c
}

func TestRunList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/public_ports", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("list: want GET, got %s", r.Method)
		}
		if got := r.Header.Get("X-Auth-Token"); got != testToken {
			t.Fatalf("list: want X-Auth-Token %q, got %q", testToken, got)
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "ports": [
    {
      "id": "port-1",
      "project_id": "proj-1",
      "network_id": "net-1",
      "ip_address": "203.0.113.10",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "description": "web frontend",
      "admin_state_up": true
    },
    {
      "id": "port-2",
      "project_id": "proj-1",
      "network_id": "net-1",
      "ip_address": "203.0.113.11",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z",
      "admin_state_up": false
    }
  ]
}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cmd, out := testCommand(t)
	if err := runList(cmd, newTestClient(server)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"port-1", "203.0.113.10", "port-2", "203.0.113.11"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunListAppliesQueryFilters(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/public_ports", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("project_id") != "proj-1" {
			t.Fatalf("list: want project_id=proj-1, got %q", q.Get("project_id"))
		}
		if q.Get("network_id") != "net-9" {
			t.Fatalf("list: want network_id=net-9, got %q", q.Get("network_id"))
		}
		if q.Get("description") != "web" {
			t.Fatalf("list: want description=web, got %q", q.Get("description"))
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{"ports": []}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cmd, _ := testCommand(t)
	for k, v := range map[string]string{
		"project-id":  "proj-1",
		"network-id":  "net-9",
		"description": "web",
	} {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatalf("set %s: %v", k, err)
		}
	}
	if err := runList(cmd, newTestClient(server)); err != nil {
		t.Fatalf("runList with filters: %v", err)
	}
}

func TestRunShow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/public_ports/port-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("show: want GET, got %s", r.Method)
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "port": {
    "id": "port-1",
    "project_id": "proj-1",
    "network_id": "net-1",
    "ip_address": "203.0.113.10",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cmd, out := testCommand(t)
	if err := runShow(cmd, newTestClient(server), "port-1"); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"port-1", "project_id: proj-1", "203.0.113.10"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunShowTable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/public_ports/port-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("show: want GET, got %s", r.Method)
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "port": {
    "id": "port-1",
    "project_id": "proj-1",
    "network_id": "net-1",
    "ip_address": "203.0.113.10",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "subnet": "203.0.113.0/24",
    "gateway": "203.0.113.1",
    "security_group_ids": ["sg-1", "sg-2"]
  }
}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("output", "table"); err != nil {
		t.Fatal(err)
	}
	if err := runShow(cmd, newTestClient(server), "port-1"); err != nil {
		t.Fatalf("runShow (table): %v", err)
	}
	// A single public port is rendered as a vertical Parameter/Value table
	// rather than a wide single-row table, and every field is visible.
	for _, want := range []string{"Parameter", "Value", "IP Address", "203.0.113.10", "Subnet", "Gateway", "sg-1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show table output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/public_ports", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("create: want POST, got %s", r.Method)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
  "port": {
    "id": "port-new",
    "project_id": "proj-1",
    "network_id": "net-1",
    "ip_address": "203.0.113.20",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("project-id", "proj-1"); err != nil {
		t.Fatal(err)
	}
	if err := runCreate(cmd, newTestClient(server)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "port-new") {
		t.Fatalf("create output missing new port; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresProjectID(t *testing.T) {
	// Isolate the config directory so no stored token leaks into the test, and
	// clear the environment fallback. runCreate must therefore fail while trying
	// to resolve the project ID, before ever touching the (nil) network client.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("SLT_KS_TOKEN", "")

	cmd, _ := testCommand(t)
	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("expected error when --project-id is missing and no token can infer it")
	}
}

func TestRunUpdate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/public_ports/port-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("update: want PATCH, got %s", r.Method)
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "port": {
    "id": "port-1",
    "project_id": "proj-1",
    "network_id": "net-1",
    "ip_address": "203.0.113.10",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "description": "new description"
  }
}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("description", "new description"); err != nil {
		t.Fatal(err)
	}
	if err := runUpdate(cmd, newTestClient(server), "port-1"); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "new description") {
		t.Fatalf("update output missing new description; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOption(t *testing.T) {
	server := httptest.NewServer(http.NewServeMux())
	t.Cleanup(server.Close)

	cmd, _ := testCommand(t)
	if err := runUpdate(cmd, newTestClient(server), "port-1"); err == nil {
		t.Fatal("expected error when no update option is provided")
	}
}

func TestRunDelete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/public_ports/port-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("delete: want DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cmd, out := testCommand(t)
	if err := runDelete(cmd, newTestClient(server), "port-1"); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "public port port-1 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}

func TestRunDeleteForce(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/public_ports/port-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("delete: want DELETE, got %s", r.Method)
		}
		if got := r.URL.Query().Get("force"); got != "true" {
			t.Fatalf("delete: want force=true, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("force", "true"); err != nil {
		t.Fatal(err)
	}
	if err := runDelete(cmd, newTestClient(server), "port-1"); err != nil {
		t.Fatalf("runDelete (force): %v", err)
	}
}
