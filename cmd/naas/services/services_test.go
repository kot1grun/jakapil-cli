/*
Copyright © 2026 Kot1gruN
*/
package services

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

// serviceListBody mirrors the JSON served by the globalrouter-go
// pkg/v1/services_test.go fixtures for a ListServices call.
const serviceListBody = `[{
	"id": "123e4567-e89b-12d3-a456-426655440000",
	"name": "vpc",
	"extension": "vpc",
	"created_at": "2170-01-01 00:00:00"
}]`

// testCommand builds a bare "services" command with every flag shared by the
// subcommands and a captured stdout buffer, so the run* helpers can be exercised
// directly.
func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "services"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetContext(context.Background())
	return cmd, &out
}

// testClient builds a real globalrouter-go V1 ServiceClient pointing at a local
// mock server. The caller registers the /services handlers they want on the
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
	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, http.MethodGet)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(serviceListBody))
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"123e4567-e89b-12d3-a456-426655440000", "vpc", "extension"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunListEmpty(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/services", func(w http.ResponseWriter, r *http.Request) {
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
