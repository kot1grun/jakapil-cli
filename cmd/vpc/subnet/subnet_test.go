/*
Copyright © 2026 Kot1gruN
*/
package subnet

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

// fakeNeutron serves a minimal networking v2 subnets API returning fixed
// records, mirroring the subset used by the VPC subnet commands.
func fakeNeutron(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(t, w, http.StatusOK, map[string]any{
				"subnets": []map[string]any{
					{"id": "sub-1", "name": "sn", "network_id": "net-1", "cidr": "10.0.0.0/24", "ip_version": 4},
					{"id": "sub-2", "name": "second", "network_id": "net-1", "cidr": "10.0.1.0/24", "ip_version": 4},
				},
			})
		case http.MethodPost:
			var body struct {
				Subnet map[string]any `json:"subnet"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.Subnet["network_id"] == "" {
				http.Error(w, "network_id required", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, http.StatusCreated, map[string]any{
				"subnet": map[string]any{
					"id": "sub-new", "name": "sn", "network_id": body.Subnet["network_id"],
					"cidr": "10.0.2.0/24", "ip_version": 4,
				},
			})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v2.0/subnets/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(t, w, http.StatusOK, map[string]any{
				"subnet": map[string]any{"id": "sub-1", "name": "sn", "network_id": "net-1", "cidr": "10.0.0.0/24", "ip_version": 4},
			})
		case http.MethodPut:
			writeJSON(t, w, http.StatusOK, map[string]any{
				"subnet": map[string]any{"id": "sub-1", "name": "renamed", "network_id": "net-1", "cidr": "10.0.0.0/24", "ip_version": 4},
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
	cmd := &cobra.Command{Use: "subnet"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.String("description", "", "")
	fs.String("network-id", "", "")
	fs.String("cidr", "", "")
	fs.String("id", "", "")
	fs.String("gateway-ip", "", "")
	fs.Int("ip-version", 4, "")
	fs.Int("limit", 0, "")
	fs.Bool("enable-dhcp", true, "")
	fs.StringSlice("dns-nameservers", nil, "")

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
	for _, want := range []string{"sub-1", "sn", "10.0.0.0/24", "sub-2"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunShow(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)

	if err := runShow(cmd, newTestClient(t, ts), []string{"sub-1"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	if !strings.Contains(out.String(), "sub-1") {
		t.Fatalf("show output missing subnet; got:\n%s", out.String())
	}
}

func TestRunCreate(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("network-id", "net-1"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("name", "created-subnet"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, newTestClient(t, ts)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "sub-new") {
		t.Fatalf("create output missing created subnet; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresNetwork(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, _ := testCommand(t)

	if err := runCreate(cmd, newTestClient(t, ts)); err == nil {
		t.Fatal("expected error when --network-id is missing")
	}
}

func TestRunCreateRejectsBadIPVersion(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("network-id", "net-1"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("ip-version", "7"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, newTestClient(t, ts)); err == nil {
		t.Fatal("expected error for unsupported IP version")
	}
}

func TestRunUpdate(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "renamed"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, newTestClient(t, ts), []string{"sub-1"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Fatalf("update output missing renamed subnet; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOptions(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, _ := testCommand(t)

	if err := runUpdate(cmd, newTestClient(t, ts), []string{"sub-1"}); err == nil {
		t.Fatal("expected error when no update option is set")
	}
}

func TestRunDelete(t *testing.T) {
	ts := fakeNeutron(t)
	cmd, out := testCommand(t)

	if err := runDelete(cmd, newTestClient(t, ts), []string{"sub-1"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "subnet sub-1 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
