/*
Copyright © 2026 Kot1gruN
*/
package publicsubnet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud"
	selvpcclient "github.com/selectel/go-selvpcclient/v5/selvpcclient"
	"github.com/selectel/go-selvpcclient/v5/selvpcclient/clients"
	clientservices "github.com/selectel/go-selvpcclient/v5/selvpcclient/clients/services"
	"github.com/spf13/cobra"
)

const testToken = "test-token"

// testCommand builds a bare command with every public-subnet flag registered and
// a captured stdout buffer.
func testCommand(t *testing.T) (*cobra.Command, *strings.Builder) {
	t.Helper()
	cmd := &cobra.Command{Use: "public-subnet"}

	fs := cmd.Flags()
	fs.String("token", "", "")
	fs.String("auth-url", "", "")
	fs.String("region", "", "")
	fs.Bool("debug", false, "")
	fs.StringP("output", "o", "yaml", "output format: json or yaml")

	// create flags
	fs.Int("prefix-length", 0, "")
	fs.String("type", "", "")

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&strings.Builder{})
	return cmd, &out
}

// testClient builds a go-selvpcclient Client pointing at a local mock server. It
// registers the identity /auth/tokens handler carrying a keystone catalog with a
// single Resell endpoint (region ru-1); the caller registers the
// /resell/v2/subnets handlers they want on the returned mux.
func testClient(t *testing.T) (*http.ServeMux, *selvpcclient.Client) {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, `{
  "token": {
    "catalog": [
      {
        "endpoints": [
          {
            "id": "e1",
            "interface": "public",
            "region_id": "ru-1",
            "region": "ru-1",
            "url": "%s"
          }
        ],
        "id": "resell-service",
        "type": "resell",
        "name": "resell"
      }
    ]
  }
}`, server.URL+"/resell")
	})

	serviceClient := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{TokenID: testToken},
		Endpoint:       server.URL + "/",
	}

	catalogService, err := clientservices.NewCatalogService(serviceClient)
	if err != nil {
		t.Fatalf("failed to init catalog service: %v", err)
	}

	requestService := clientservices.NewRequestService(serviceClient)

	return mux, &selvpcclient.Client{
		Resell: clients.NewResellClient(requestService, catalogService, "ru-1"),
	}
}

func TestRunList(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/resell/v2/subnets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("list: want GET, got %s", r.Method)
		}
		if got := r.Header.Get("X-Auth-Token"); got != testToken {
			t.Fatalf("list: want X-Auth-Token %q, got %q", testToken, got)
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "subnets": [
    {
      "cidr": "203.0.113.0/24",
      "id": 112233,
      "project_id": "49338ac045f448e294b25d013f890317",
      "region": "ru-3",
      "status": "ACTIVE"
    },
    {
      "cidr": "198.51.100.0/24",
      "id": 112234,
      "project_id": "9c97bdc75295493096cf5edcb8c37933",
      "region": "ru-2",
      "status": "ACTIVE"
    }
  ]
}`)
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"112233", "112234", "203.0.113.0/24", "198.51.100.0/24", "ru-3", "ru-2"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunListTable(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/resell/v2/subnets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "subnets": [
    {
      "cidr": "203.0.113.0/24",
      "id": 112233,
      "project_id": "49338ac045f448e294b25d013f890317",
      "region": "ru-3",
      "status": "ACTIVE"
    }
  ]
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("output", "table"); err != nil {
		t.Fatal(err)
	}

	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"ID", "Status", "Region", "CIDR", "Network ID", "112233", "ACTIVE", "ru-3", "203.0.113.0/24", "unassigned"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list table output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreate(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/resell/v2/subnets/projects/9c97bdc75295493096cf5edcb8c37933", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("create: want POST, got %s", r.Method)
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "subnets": [
    {
      "cidr": "203.0.113.0/29",
      "id": 112233,
      "project_id": "9c97bdc75295493096cf5edcb8c37933",
      "region": "ru-2",
      "status": "DOWN"
    }
  ]
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("region", "ru-2"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("type", "ipv4"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("prefix-length", "29"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, cl, "9c97bdc75295493096cf5edcb8c37933"); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	for _, want := range []string{"112233", "203.0.113.0/29", "ru-2"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("create output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreateAcceptsIPv6Type(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/resell/v2/subnets/projects/p1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "subnets": [
    {
      "id": 1,
      "region": "ru-2",
      "status": "DOWN"
    }
  ]
}`)
	})

	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("region", "ru-2"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("type", "IPv6 "); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("prefix-length", "64"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, cl, "p1"); err != nil {
		t.Fatalf("runCreate (IPv6 upper-cased/spaced): %v", err)
	}
}

func TestRunCreateRequiresRegion(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("type", "ipv4"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("prefix-length", "29"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, nil, "p1"); err == nil {
		t.Fatal("expected error when --region is missing")
	}
}

func TestRunCreateRejectsBadType(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("region", "ru-2"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("type", "ipv7"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("prefix-length", "29"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, nil, "p1"); err == nil {
		t.Fatal("expected error for unsupported IP version")
	}
}

func TestRunCreateRejectsPrefixLength(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("region", "ru-2"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("type", "ipv4"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("prefix-length", "0"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, nil, "p1"); err == nil {
		t.Fatal("expected error for invalid prefix length")
	}
}

func TestRunCreateInfersProjectIDFromToken(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// Identity handler: returns both the service catalog (consumed by
	// NewResellClient) and the target project (consumed by resolveProjectID).
	// go-selvpcclient requests /auth/tokens while gophercloud's identity
	// AuthenticateV3 requests /v3/auth/tokens, so both routes share the body.
	identity := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, `{
  "token": {
    "project": {
      "id": "9c97bdc75295493096cf5edcb8c37933",
      "domain": {"id": "dom1", "name": "default"},
      "name": "big-project"
    },
    "catalog": [
      {
        "endpoints": [
          {
            "id": "e1",
            "interface": "public",
            "region_id": "ru-1",
            "region": "ru-1",
            "url": "%s"
          }
        ],
        "id": "resell-service",
        "type": "resell",
        "name": "resell"
      }
    ]
  }
}`, server.URL+"/resell")
	}
	mux.HandleFunc("/auth/tokens", identity)
	mux.HandleFunc("/v3/auth/tokens", identity)

	// Create endpoint matched by the project ID inferred from the token.
	mux.HandleFunc("/resell/v2/subnets/projects/9c97bdc75295493096cf5edcb8c37933", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "subnets": [
    {
      "id": 7,
      "region": "ru-1",
      "status": "DOWN"
    }
  ]
}`)
	})

	cmd, out := testCommand(t)
	for k, v := range map[string]string{
		"token":         testToken,
		"auth-url":      server.URL,
		"region":        "ru-1",
		"type":          "ipv4",
		"prefix-length": "29",
	} {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatalf("set %s: %v", k, err)
		}
	}

	cl, err := NewResellClient(cmd)
	if err != nil {
		t.Fatalf("NewResellClient: %v", err)
	}

	// projectID is empty -> must be inferred from the token scope.
	if err := runCreate(cmd, cl, ""); err != nil {
		t.Fatalf("runCreate (auto project ID): %v", err)
	}
	if !strings.Contains(out.String(), "id: 7") {
		t.Fatalf("create output missing created subnet; got:\n%s", out.String())
	}
}

func TestRunCreateRejectsUnscopedTokenForProjectInference(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// Identity handler returns the catalog but NO project scope. go-selvpcclient
	// requests /auth/tokens while gophercloud requests /v3/auth/tokens.
	identity := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprintf(w, `{
  "token": {
    "catalog": [
      {
        "endpoints": [
          {
            "id": "e1",
            "interface": "public",
            "region_id": "ru-1",
            "region": "ru-1",
            "url": "%s"
          }
        ],
        "id": "resell-service",
        "type": "resell",
        "name": "resell"
      }
    ]
  }
}`, server.URL+"/resell")
	}
	mux.HandleFunc("/auth/tokens", identity)
	mux.HandleFunc("/v3/auth/tokens", identity)

	cmd, _ := testCommand(t)
	for k, v := range map[string]string{
		"token":         testToken,
		"auth-url":      server.URL,
		"region":        "ru-1",
		"type":          "ipv4",
		"prefix-length": "29",
	} {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatalf("set %s: %v", k, err)
		}
	}

	cl, err := NewResellClient(cmd)
	if err != nil {
		t.Fatalf("NewResellClient: %v", err)
	}

	if err := runCreate(cmd, cl, ""); err == nil {
		t.Fatal("expected error when the token has no project scope and no project ID is given")
	}
}

func TestRunDelete(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/resell/v2/subnets/112233", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("delete: want DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, cl, "112233"); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "public subnet 112233 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
