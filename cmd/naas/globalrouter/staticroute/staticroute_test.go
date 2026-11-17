/*
Copyright © 2026 Kot1gruN
*/
package staticroute

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

const staticRouteID = "33333333-3333-3333-3333-333333333333"
const routerID = "11111111-1111-1111-1111-111111111111"
const subnetID = "22222222-2222-2222-2222-222222222222"

// staticRouteBody mirrors the JSON served by the globalrouter-go
// pkg/v1/static_routes_test.go fixtures for a single StaticRoute.
const staticRouteBody = `{
	"id": "33333333-3333-3333-3333-333333333333",
	"name": "stat_route",
	"cidr": "0.0.0.0/0",
	"next_hop": "10.20.0.252",
	"router_id": "11111111-1111-1111-1111-111111111111",
	"created_at": "2025-10-22T09:22:37.294081",
	"updated_at": "2025-10-22T09:22:48.728094",
	"status": "ACTIVE",
	"account_id": "777777",
	"project_id": null,
	"subnet_id": "22222222-2222-2222-2222-222222222222",
	"netops_static_route_id": "33333333-3333-3333-3333-333333333333",
	"tags": ["net"]
}`

// staticRouteListBody is the static route fixture wrapped in an array, as served
// by StaticRoutes.
const staticRouteListBody = `[` + staticRouteBody + `]`

// testCommand builds a bare "static-route" command with every flag shared by the
// subcommands and a captured stdout buffer, so the run* helpers can be exercised
// directly.
func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "static-route"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.String("router-id", "", "")
	fs.String("next-hop", "", "")
	fs.String("cidr", "", "")
	fs.String("project-id", "", "")
	fs.StringSlice("tags", nil, "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	return cmd, &out
}

// testClient builds a real globalrouter-go V1 ServiceClient pointing at a local
// mock server. The caller registers the /static_routes handlers they want on the
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
	mux.HandleFunc("/static_routes", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(staticRouteListBody))
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{staticRouteID, "stat_route", routerID, "10.20.0.252", "0.0.0.0/0", subnetID} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

const otherRouterID = "44444444-4444-4444-4444-444444444444"

// multiRouterBody is a list fixture with two static routes attached to different
// routers, used to exercise the client-side --router-id filter.
const multiRouterBody = `[
	{"id":"33333333-3333-3333-3333-333333333333","name":"stat_route","cidr":"0.0.0.0/0","next_hop":"10.20.0.252","router_id":"` + routerID + `","subnet_id":"` + subnetID + `","status":"ACTIVE","tags":["net"]},
	{"id":"55555555-5555-5555-5555-555555555555","name":"other_route","cidr":"10.0.0.0/8","next_hop":"10.20.0.253","router_id":"` + otherRouterID + `","subnet_id":"` + subnetID + `","status":"ACTIVE","tags":[]}
]`

func TestRunListRouterFilter(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/static_routes", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(multiRouterBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("router-id", routerID)
	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if !strings.Contains(out.String(), "stat_route") {
		t.Fatalf("filtered output missing matching route; got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "other_route") {
		t.Fatalf("filtered output unexpectedly contains a route of another router; got:\n%s", out.String())
	}
}

func TestRunListEmpty(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/static_routes", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/static_routes/"+staticRouteID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(staticRouteBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("output", "table")
	if err := runShow(cmd, cl, staticRouteID); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"Router ID", "Next Hop", "CIDR", "10.20.0.252", "0.0.0.0/0", "Subnet ID"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreate(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/static_routes", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPost)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(staticRouteBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("router-id", routerID)
	_ = cmd.Flags().Set("next-hop", "10.20.0.252")
	_ = cmd.Flags().Set("cidr", "0.0.0.0/0")
	if err := runCreate(cmd, cl); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), staticRouteID) {
		t.Fatalf("create output missing ID; got:\n%s", out.String())
	}
}

func TestRunCreateMissingRequired(t *testing.T) {
	cmd, _ := testCommand(t)
	mux, cl := testClient(t)
	_ = mux
	_ = cmd.Flags().Set("router-id", routerID)

	if err := runCreate(cmd, cl); err == nil {
		t.Fatal("expected error for missing next-hop, got nil")
	}
}

func TestRunUpdate(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/static_routes/"+staticRouteID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPut)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(staticRouteBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("tags", "net")
	if err := runUpdate(cmd, cl, staticRouteID); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), staticRouteID) {
		t.Fatalf("update output missing ID; got:\n%s", out.String())
	}
}

func TestRunUpdateNoOptions(t *testing.T) {
	cmd, _ := testCommand(t)
	mux, cl := testClient(t)
	_ = mux

	if err := runUpdate(cmd, cl, staticRouteID); err == nil {
		t.Fatal("expected error when no update option given, got nil")
	}
}

func TestRunDelete(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/static_routes/"+staticRouteID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, cl, staticRouteID); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
