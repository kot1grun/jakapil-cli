/*
Copyright © 2026 Kot1gruN
*/
package network

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

// fakeNeutron serves a minimal networking v2 API (the subset used by the VPC
// network commands) returning fixed network records. It asserts the HTTP method
// and, for create/update, that the request body carries the expected fields.
func fakeNeutron(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(t, w, http.StatusOK, map[string]any{
				"networks": []map[string]any{
					{"id": "net-1", "name": "nf", "status": "ACTIVE"},
					{"id": "net-2", "name": "second", "status": "DOWN"},
				},
			})
		case http.MethodPost:
			var body struct {
				Network map[string]any `json:"network"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.Network["name"] == "" {
				http.Error(w, "name required", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"network": map[string]any{
					"id": "net-new", "name": body.Network["name"], "status": "ACTIVE",
				},
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2.0/networks/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(t, w, http.StatusOK, map[string]any{
				"network": map[string]any{"id": "net-1", "name": "nf", "status": "ACTIVE"},
			})
		case http.MethodPut:
			writeJSON(t, w, http.StatusOK, map[string]any{
				"network": map[string]any{"id": "net-1", "name": "renamed", "status": "ACTIVE"},
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

// newTestClient builds a gophercloud networking ServiceClient pointed at ts.
func newTestClient(t *testing.T, ts *httptest.Server) *gophercloud.ServiceClient {
	t.Helper()
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{},
		Endpoint:       ts.URL + "/",
		ResourceBase:   ts.URL + "/v2.0/",
	}
}

// testCommand builds a bare cobra command carrying the flags the network run
// functions read, with an --output flag and a captured stdout.
func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "network"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.String("description", "", "")
	fs.String("status", "", "")
	fs.String("id", "", "")
	fs.Int("limit", 0, "")
	fs.Bool("admin-state-up", true, "")
	fs.Bool("shared", false, "")

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
	if !strings.Contains(out.String(), "net-1") || !strings.Contains(out.String(), "nf") {
		t.Fatalf("list output missing network; got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "net-2") {
		t.Fatalf("list output missing second network; got:\n%s", out.String())
	}
}

func TestRunShow(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)

	if err := runShow(cmd, newTestClient(t, ts), []string{"net-1"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	if !strings.Contains(out.String(), "net-1") {
		t.Fatalf("show output missing network; got:\n%s", out.String())
	}
}

func TestRunCreate(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "created-net"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, newTestClient(t, ts)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "net-new") || !strings.Contains(out.String(), "created-net") {
		t.Fatalf("create output missing created network; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresName(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, _ := testCommand(t)

	if err := runCreate(cmd, newTestClient(t, ts)); err == nil {
		t.Fatal("expected error when --name is missing")
	}
}

func TestRunUpdate(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "renamed"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, newTestClient(t, ts), []string{"net-1"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Fatalf("update output missing renamed network; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOptions(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, _ := testCommand(t)

	if err := runUpdate(cmd, newTestClient(t, ts), []string{"net-1"}); err == nil {
		t.Fatal("expected error when no update option is set")
	}
}

func TestRunDelete(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)

	if err := runDelete(cmd, newTestClient(t, ts), []string{"net-1"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "network net-1 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
