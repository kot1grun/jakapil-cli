/*
Copyright © 2026 Kot1gruN
*/
package subnet

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"
)

const testToken = "test-token"

const subnetID = "33333333-3333-3333-3333-333333333333"
const networkID = "44444444-4444-4444-4444-444444444444"
const osSubnetID = "55555555-5555-5555-5555-555555555555"

// subnetBody mirrors the JSON served by the globalrouter-go
// pkg/v1/subnets_test.go fixtures for a single Subnet.
const subnetBody = `{
	"id": "33333333-3333-3333-3333-333333333333",
	"name": "subnet_cloud",
	"gateway": "10.0.14.1",
	"cidr": "10.0.14.0/24",
	"network_id": "44444444-4444-4444-4444-444444444444",
	"updated_at": "2025-12-12T09:57:48.609636",
	"created_at": "2025-12-12T09:57:27.524285",
	"status": "ACTIVE",
	"account_id": "777777",
	"project_id": null,
	"service_addresses": [
		"10.0.14.253",
		"10.0.14.254"
	],
	"os_subnet_id": "55555555-5555-5555-5555-555555555555",
	"sv_subnet_id": "77777777-7777-7777-7777-777777777777",
	"netops_subnet_id": "33333333-3333-3333-3333-333333333333",
	"tags": ["tag2"]
}`

// subnetListBody is the subnet fixture wrapped in an array, as served by Subnets.
const subnetListBody = `[` + subnetBody + `]`

// testCommand builds a bare "subnet" command with every flag shared by the
// subcommands and a captured stdout buffer, so the run* helpers can be exercised
// directly.
func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "subnet"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.String("network-id", "", "")
	fs.String("cidr", "", "")
	fs.String("gateway", "", "")
	fs.String("os-subnet-id", "", "")
	fs.String("project-id", "", "")
	fs.StringSlice("service-addresses", nil, "")
	fs.StringSlice("tags", nil, "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	return cmd, &out
}

// testClient builds a real globalrouter-go V1 ServiceClient pointing at a local
// mock server. The caller registers the /subnets handlers they want on the
// returned mux.
func testClient(t *testing.T) (*http.ServeMux, *v1.ServiceClient) {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := v1.NewClientV1(testToken, v1.WithAPIUrl(server.URL))
	if err != nil {
		t.Fatalf("init globalrouter client: %v", err)
	}
	return mux, client
}

func verifyRequest(t *testing.T, r *http.Request, method string) {
	t.Helper()
	if r.Method != method {
		t.Fatalf("want method %s, got %s", method, r.Method)
	}
	if got := r.Header.Get("X-Auth-Token"); got != testToken {
		t.Fatalf("want X-Auth-Token %q, got %q", testToken, got)
	}
}

func TestRunList(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/subnets", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(subnetListBody))
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{subnetID, "subnet_cloud", networkID, "10.0.14.1", "10.0.14.0/24"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunListEmptyTable(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/subnets", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("output", "table")
	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if !strings.Contains(out.String(), "No resources found") {
		t.Fatalf("empty list output missing notice; got:\n%s", out.String())
	}
}

func TestRunShowTable(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/subnets/"+subnetID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(subnetBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("output", "table")
	if err := runShow(cmd, cl, subnetID); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"Network ID", "Gateway", "CIDR", "10.0.14.1", "10.0.14.0/24", "OS Subnet ID"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreateDedicated(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/subnets", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPost)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(subnetBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("network-id", networkID)
	_ = cmd.Flags().Set("cidr", "10.0.14.0/24")
	if err := runCreate(cmd, cl); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), subnetID) {
		t.Fatalf("create output missing ID; got:\n%s", out.String())
	}
}

func TestRunCreateVPC(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/subnets", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPost)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(subnetBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("network-id", networkID)
	_ = cmd.Flags().Set("cidr", "10.0.14.0/24")
	_ = cmd.Flags().Set("os-subnet-id", osSubnetID)
	if err := runCreate(cmd, cl); err != nil {
		t.Fatalf("runCreate (VPC): %v", err)
	}
	if !strings.Contains(out.String(), subnetID) {
		t.Fatalf("VPC create output missing ID; got:\n%s", out.String())
	}
}

func TestRunCreateMissingRequired(t *testing.T) {
	_, cl := testClient(t)
	cmd, _ := testCommand(t)

	if err := runCreate(cmd, cl); err == nil {
		t.Fatal("expected error for missing network-id and cidr, got nil")
	}
}

func TestRunUpdate(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/subnets/"+subnetID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPut)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(subnetBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("tags", "tag2")
	if err := runUpdate(cmd, cl, subnetID); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), subnetID) {
		t.Fatalf("update output missing ID; got:\n%s", out.String())
	}
}

func TestRunUpdateNoOptions(t *testing.T) {
	_, cl := testClient(t)
	cmd, _ := testCommand(t)

	if err := runUpdate(cmd, cl, subnetID); err == nil {
		t.Fatal("expected error when no update option given, got nil")
	}
}

func TestRunDelete(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/subnets/"+subnetID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodDelete)
		if q := r.URL.Query().Get("disconnect"); q != "true" {
			t.Fatalf("delete: want disconnect=true, got %q", q)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, cl, subnetID); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "disconnected") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
