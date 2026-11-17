/*
Copyright © 2026 Kot1gruN
*/
package globalrouter

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

// routerListBody mirrors the JSON served by the globalrouter-go
// pkg/v1/routers_test.go fixtures for a Routers call.
const routerListBody = `[{
	"id": "11111111-1111-1111-1111-111111111111",
	"name": "test_rtr",
	"updated_at": "",
	"created_at": "2025-11-16T08:27:19.076775",
	"status": "ACTIVE",
	"enabled": true,
	"account_id": "777777",
	"project_id": null,
	"netops_router_id": "11111111-1111-1111-1111-111111111111",
	"tags": ["tag1"],
	"leak_uuid": null,
	"prefix_pool_id": "88a5f82b-3151-4d84-b111-2e06ee44d899",
	"vpn_id": 1,
	"networks": [],
	"prefix_pool": {"id": null, "prefixes": null, "default": null}
}]`

// routerBody is the single-router JSON served for create/update responses,
// mirroring the globalrouter-go router fixtures.
const routerBody = `{
	"id": "11111111-1111-1111-1111-111111111111",
	"name": "test_rtr",
	"updated_at": "2025-11-16T08:27:19.782861",
	"created_at": "2025-11-16T08:27:19.076775",
	"status": "ACTIVE",
	"enabled": true,
	"account_id": "777777",
	"project_id": null,
	"netops_router_id": "11111111-1111-1111-1111-111111111111",
	"tags": ["blue", "red"],
	"leak_uuid": null,
	"prefix_pool_id": "88a5f82b-3151-4d84-b111-2e06ee44d899",
	"vpn_id": 1,
	"networks": [],
	"prefix_pool": {"id": null, "prefixes": null, "default": null}
}`

// testCommand builds a bare "router" command with every flag shared by the
// subcommands and a captured stdout buffer, so the run* helpers can be
// exercised directly.
func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "router"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.StringSlice("tags", nil, "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	return cmd, &out
}

// testClient builds a real globalrouter-go V1 ServiceClient pointing at a local
// mock server. The caller registers the /routers handlers they want on the
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
	mux.HandleFunc("/routers", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(routerListBody))
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"11111111-1111-1111-1111-111111111111", "test_rtr", "ACTIVE", "tag1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunListEmpty(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/routers", func(w http.ResponseWriter, r *http.Request) {
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

func TestRunCreate(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/routers", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPost)
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("create: want Content-Type application/json, got %q", ct)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(routerBody))
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "test_rtr"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("tags", "blue,red"); err != nil {
		t.Fatal(err)
	}
	if err := runCreate(cmd, cl); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "11111111-1111-1111-1111-111111111111") {
		t.Fatalf("create output missing router id; got:\n%s", out.String())
	}
}
func TestRunCreateRequiresName(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate without a name succeeded; want an error")
	}
}

func TestRunShowTable(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	mux, cl := testClient(t)
	mux.HandleFunc("/routers/"+id, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(routerBody))
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("output", "table")
	if err := runShow(cmd, cl, id); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"ID", "Name", "test_rtr", "ACTIVE", "Account ID", "777777", "Prefix Pool ID", "88a5f82b-3151-4d84-b111-2e06ee44d899"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunUpdate(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	mux, cl := testClient(t)
	mux.HandleFunc("/routers/"+id, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodPut)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(routerBody))
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "test_rtr"); err != nil {
		t.Fatal(err)
	}
	if err := runUpdate(cmd, cl, id); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "test_rtr") {
		t.Fatalf("update output missing updated name; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOption(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runUpdate(cmd, nil, "router-1"); err == nil {
		t.Fatal("runUpdate without any option succeeded; want an error")
	}
}

func TestRunDelete(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	mux, cl := testClient(t)
	mux.HandleFunc("/routers/"+id, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodDelete)
		w.WriteHeader(http.StatusNoContent)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, cl, id); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), id) {
		t.Fatalf("delete output missing router id; got:\n%s", out.String())
	}
}
