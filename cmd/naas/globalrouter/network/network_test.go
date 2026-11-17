/*
Copyright © 2026 Kot1gruN
*/
package network

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

const networkID = "11111111-1111-1111-1111-111111111111"
const dedicatedNetID = "88888888-8888-8888-8888-888888888888"
const routerID = "22222222-2222-2222-2222-222222222222"
const zoneID = "33333333-3333-3333-3333-333333333333"
const osNetworkID = "55555555-5555-5555-5555-555555555555"

// vpcNetworkBody mirrors the VPC network JSON served by the globalrouter-go
// pkg/v1/networks_test.go fixtures for a VPCNetworkCreate response.
const vpcNetworkBody = `{
	"id": "11111111-1111-1111-1111-111111111111",
	"name": "cloud_net",
	"vlan": 21412,
	"router_id": "22222222-2222-2222-2222-222222222222",
	"zone_id": "33333333-3333-3333-3333-333333333333",
	"project_id": "77777777777777777777777777777777",
	"vdc_name": null,
	"updated_at": null,
	"created_at": "2025-12-08T18:57:57.412757",
	"status": "ACTIVE",
	"account_id": "777777",
	"sv_network_id": "44444444-4444-4444-4444-444444444444",
	"netops_vlan_uuid": "11111111-1111-1111-1111-111111111111",
	"tags": ["tag11"],
	"inner_vlan": null,
	"os_network_id": "55555555-5555-5555-5555-555555555555"
}`

// dedicatedNetworkBody mirrors the dedicated network JSON served by the
// globalrouter-go networks_test.go fixtures for a DedicatedNetworkCreate
// response.
const dedicatedNetworkBody = `{
	"id": "88888888-8888-8888-8888-888888888888",
	"name": "dedicated_net",
	"vlan": 16461,
	"router_id": "22222222-2222-2222-2222-222222222222",
	"zone_id": "99999999-9999-9999-9999-999999999999",
	"project_id": null,
	"vdc_name": null,
	"updated_at": "2025-12-05T17:00:25.690283",
	"created_at": "2025-12-05T17:00:13.637735",
	"status": "ACTIVE",
	"account_id": "777777",
	"sv_network_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	"netops_vlan_uuid": "88888888-8888-8888-8888-888888888888",
	"tags": ["tag22"],
	"inner_vlan": 321,
	"os_network_id": null
}`

// networkListBody is a network fixture wrapped in an array, as served by
// Networks.
const networkListBody = `[` + vpcNetworkBody + `]`

// testCommand builds a bare "network" command with every flag shared by the
// subcommands and a captured stdout buffer, so the run* helpers can be exercised
// directly.
func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "network"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.String("router-id", "", "")
	fs.String("zone-id", "", "")
	fs.String("type", "vpc", "")
	fs.String("os-network-id", "", "")
	fs.String("project-id", "", "")
	fs.Int("vlan", 0, "")
	fs.Int("inner-vlan", 0, "")
	fs.StringSlice("tags", nil, "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	return cmd, &out
}

// testClient builds a real globalrouter-go V1 ServiceClient pointing at a local
// mock server. The caller registers the /networks handlers they want on the
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
	mux.HandleFunc("/networks", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(networkListBody))
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{networkID, "cloud_net", routerID, zoneID, "ACTIVE", osNetworkID, "tag11"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunListEmpty(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/networks", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/networks/"+networkID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(vpcNetworkBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("output", "table")
	if err := runShow(cmd, cl, networkID); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"Zone ID", "Os Network ID", "VLAN", "21412", "Sv Network ID"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreateVPC(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/networks", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPost)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(vpcNetworkBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("router-id", routerID)
	_ = cmd.Flags().Set("zone-id", zoneID)
	_ = cmd.Flags().Set("os-network-id", osNetworkID)
	if err := runCreate(cmd, cl); err != nil {
		t.Fatalf("runCreate (vpc): %v", err)
	}
	if !strings.Contains(out.String(), networkID) {
		t.Fatalf("create output missing ID; got:\n%s", out.String())
	}
}

func TestRunCreateDedicated(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/networks", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPost)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(dedicatedNetworkBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("type", "dedicated")
	_ = cmd.Flags().Set("router-id", routerID)
	_ = cmd.Flags().Set("zone-id", "99999999-9999-9999-9999-999999999999")
	_ = cmd.Flags().Set("vlan", "16461")
	if err := runCreate(cmd, cl); err != nil {
		t.Fatalf("runCreate (dedicated): %v", err)
	}
	if !strings.Contains(out.String(), dedicatedNetID) {
		t.Fatalf("create output missing ID; got:\n%s", out.String())
	}
}

func TestRunCreateMissingRequired(t *testing.T) {
	cmd, _ := testCommand(t)
	mux, cl := testClient(t)
	_ = mux
	_ = cmd.Flags().Set("router-id", routerID)

	if err := runCreate(cmd, cl); err == nil {
		t.Fatal("expected error for missing zone-id, got nil")
	}
}

func TestRunCreateBadType(t *testing.T) {
	cmd, _ := testCommand(t)
	mux, cl := testClient(t)
	_ = mux
	_ = cmd.Flags().Set("router-id", routerID)
	_ = cmd.Flags().Set("zone-id", zoneID)
	_ = cmd.Flags().Set("type", "bogus")

	if err := runCreate(cmd, cl); err == nil {
		t.Fatal("expected error for unknown network type, got nil")
	}
}

func TestRunUpdate(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/networks/"+networkID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPut)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(vpcNetworkBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "cloud_net")
	if err := runUpdate(cmd, cl, networkID); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "cloud_net") {
		t.Fatalf("update output missing name; got:\n%s", out.String())
	}
}

func TestRunUpdateNoOptions(t *testing.T) {
	cmd, _ := testCommand(t)
	mux, cl := testClient(t)
	_ = mux

	if err := runUpdate(cmd, cl, networkID); err == nil {
		t.Fatal("expected error when no update option given, got nil")
	}
}

func TestRunDelete(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/networks/"+networkID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, cl, networkID); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "disconnected") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
