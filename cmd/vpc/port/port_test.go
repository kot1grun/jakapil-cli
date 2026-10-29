/*
Copyright © 2026 Kot1gruN
*/
package port

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/spf13/cobra"
)

// fakeNeutron serves a minimal networking v2 ports API returning fixed
// records, mirroring the subset used by the VPC port commands.
func fakeNeutron(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(t, w, http.StatusOK, map[string]any{
				"ports": []map[string]any{
					{"id": "port-1", "name": "pf", "network_id": "net-1", "status": "ACTIVE"},
					{"id": "port-2", "name": "second", "network_id": "net-1", "status": "DOWN"},
				},
			})
		case http.MethodPost:
			var body struct {
				Port map[string]any `json:"port"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.Port["network_id"] == "" {
				http.Error(w, "network_id required", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"port": map[string]any{
					"id": "port-new", "name": "pf", "network_id": body.Port["network_id"], "status": "DOWN",
				},
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2.0/ports/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			writeJSON(t, w, http.StatusOK, map[string]any{
				"port": map[string]any{"id": "port-1", "name": "renamed", "network_id": "net-1", "status": "ACTIVE"},
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newTestClient(t *testing.T, ts *httptest.Server) *gophercloud.ServiceClient {
	t.Helper()
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       ts.URL + "/",
		ResourceBase:   ts.URL + "/v2.0/",
	}
}

func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "port"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.String("description", "", "")
	fs.String("network-id", "", "")
	fs.String("status", "", "")
	fs.String("mac-address", "", "")
	fs.String("device-id", "", "")
	fs.String("device-owner", "", "")
	fs.String("id", "", "")
	fs.Int("limit", 0, "")
	fs.Bool("admin-state-up", true, "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunList(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)

	if err := runList(cmd, newTestClient(t, ts)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"port-1", "pf", "port-2"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreate(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("network-id", "net-1"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("name", "created-port"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, newTestClient(t, ts)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "port-new") {
		t.Fatalf("create output missing created port; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresNetwork(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, _ := testCommand(t)

	if err := runCreate(cmd, newTestClient(t, ts)); err == nil {
		t.Fatal("expected error when --network-id is missing")
	}
}

func TestRunUpdate(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "renamed"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, newTestClient(t, ts), []string{"port-1"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Fatalf("update output missing renamed port; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOptions(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, _ := testCommand(t)

	if err := runUpdate(cmd, newTestClient(t, ts), []string{"port-1"}); err == nil {
		t.Fatal("expected error when no update option is set")
	}
}

func TestRunDelete(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)

	if err := runDelete(cmd, newTestClient(t, ts), []string{"port-1"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "port port-1 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
