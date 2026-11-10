/*
Copyright © 2026 Kot1gruN
*/
package selproject

import (
	"errors"
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

// testCommand builds a bare command with every selproject flag registered and a
// captured stdout buffer.
func testCommand(t *testing.T) (*cobra.Command, *strings.Builder) {
	t.Helper()
	cmd := &cobra.Command{Use: "selproject"}

	fs := cmd.Flags()
	fs.String("token", "", "")
	fs.String("auth-url", "", "")
	fs.String("region", "", "")
	fs.Bool("debug", false, "")
	fs.StringP("output", "o", "yaml", "output format: json or yaml")

	// create flags
	fs.String("name", "", "")
	fs.Bool("skip-quotas-init", false, "")

	// update flags
	fs.String("custom-url", "", "")
	fs.String("theme-color", "", "")
	fs.String("theme-logo", "", "")

	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&strings.Builder{})
	return cmd, &out
}

// testClient builds a go-selvpcclient Client, mirroring how a real account needs
// one, but pointing at the local mock server. It registers the identity
// /auth/tokens handler carrying a keystone catalog with a single Resell endpoint
// (region ru-1), and the caller registers the /resell/v2/projects handlers they
// want on the returned mux.
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
	mux.HandleFunc("/resell/v2/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("list: want GET, got %s", r.Method)
		}
		if got := r.Header.Get("X-Auth-Token"); got != testToken {
			t.Fatalf("list: want X-Auth-Token %q, got %q", testToken, got)
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "projects": [
    {
      "custom_url": null,
      "enabled": true,
      "id": "49338ac045f448e294b25d013f890317",
      "name": "Project1",
      "url": "https://xxxxxx.selvpc.ru"
    },
    {
      "custom_url": "https://custom.example",
      "enabled": false,
      "id": "9c97bdc75295493096cf5edcb8c37933",
      "name": "Project2",
      "url": "https://yyyyyy.selvpc.ru"
    }
  ]
}`)
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, cl); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"49338ac045f448e294b25d013f890317", "Project1", "9c97bdc75295493096cf5edcb8c37933", "Project2"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunListError(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/resell/v2/projects", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"bad"}`, http.StatusBadGateway)
	})

	cmd, _ := testCommand(t)
	if err := runList(cmd, cl); err == nil {
		t.Fatal("runList against a failing endpoint succeeded; want an error")
	}
}

func TestRunCreate(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/resell/v2/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("create: want POST, got %s", r.Method)
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "project": {
    "enabled": true,
    "id": "9c97bdc75295493096cf5edcb8c37933",
    "name": "Project2",
    "url": "https://yyyyyy.selvpc.ru"
  }
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "Project2")
	if err := runCreate(cmd, cl); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "9c97bdc75295493096cf5edcb8c37933") {
		t.Fatalf("create output missing project id; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresName(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runCreate(cmd, nil); err == nil {
		t.Fatal("runCreate without a name succeeded; want an error")
	}
}

func TestRunUpdate(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/resell/v2/projects/f9ede488e5f14bac8962d8c53d0af9f4", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("update: want PATCH, got %s", r.Method)
		}
		w.Header().Add("Content-Type", "application/json")
		fmt.Fprint(w, `{
  "project": {
    "custom_url": null,
    "enabled": true,
    "id": "f9ede488e5f14bac8962d8c53d0af9f4",
    "name": "Project3",
    "theme": {"logo": null, "color": "#581845"},
    "url": "https://zzzzzz.selvpc.ru"
  }
}`)
	})

	cmd, out := testCommand(t)
	_ = cmd.Flags().Set("name", "Project3")
	if err := runUpdate(cmd, cl, "f9ede488e5f14bac8962d8c53d0af9f4"); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "Project3") {
		t.Fatalf("update output missing updated name; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOption(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runUpdate(cmd, nil, "f9ede488e5f14bac8962d8c53d0af9f4"); err == nil {
		t.Fatal("runUpdate without any option succeeded; want an error")
	}
}

func TestRunDelete(t *testing.T) {
	mux, cl := testClient(t)
	mux.HandleFunc("/resell/v2/projects/9c97bdc75295493096cf5edcb8c37933", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("delete: want DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, cl, "9c97bdc75295493096cf5edcb8c37933"); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "9c97bdc75295493096cf5edcb8c37933") {
		t.Fatalf("delete output missing project id; got:\n%s", out.String())
	}
}

func TestResellErrorAccountScope(t *testing.T) {
	// The "Scope.account scope token" policy signature must produce an
	// actionable hint pointing at the pool-specific base URL, while the
	// original gophercloud error stays intact so callers can still match it.
	base := gophercloud.ErrUnexpectedResponseCode{
		URL:    "https://api.selectel.ru/vpc/resell/v2/projects",
		Method: "GET",
		Actual: 403,
		Body:   []byte(`{"error": "policy_does_not_allow_this_request", "message": "not allowed with Scope.account scope token"}`),
	}

	err := resellError(base)
	if err == nil {
		t.Fatal("resellError returned nil for a policy error")
	}
	if !strings.Contains(err.Error(), "docs.selectel.ru/api/urls/") {
		t.Fatalf("error misses the base-URL hint; got:\n%s", err.Error())
	}

	// The wrapped error still unwraps to the original gophercloud error.
	var httpErr gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &httpErr) {
		t.Fatalf("cannot unwrap to ErrUnexpectedResponseCode; got %T", err)
	}
	if httpErr.Actual != 403 {
		t.Fatalf("unwrapped status = %d, want 403", httpErr.Actual)
	}
	if !strings.Contains(string(httpErr.Body), "policy_does_not_allow_this_request") {
		t.Fatalf("unwrapped body lost the policy signature; got %q", httpErr.Body)
	}
}

func TestResellErrorUnrelatedPassThrough(t *testing.T) {
	// Errors without the account-scope policy signature must be returned
	// unchanged so that formatting and type assertions of other failures work.
	base := gophercloud.ErrUnexpectedResponseCode{
		URL:    "https://example.test/v2/projects",
		Method: "GET",
		Actual: 502,
		Body:   []byte(`{"message": "upstream down"}`),
	}

	got := resellError(base)
	if got == nil {
		t.Fatal("resellError returned nil for an unrelated error")
	}
	// The unchanged error must still unwrap to the same status/body.
	var httpErr gophercloud.ErrUnexpectedResponseCode
	if !errors.As(got, &httpErr) {
		t.Fatalf("resellError changed the error type: got %T", got)
	}
	if httpErr.Actual != 502 {
		t.Fatalf("unwrapped status = %d, want 502", httpErr.Actual)
	}
	if string(httpErr.Body) != string(base.Body) {
		t.Fatalf("unrelated error body was rewritten; got %q", httpErr.Body)
	}
}

func TestResellErrorNil(t *testing.T) {
	if err := resellError(nil); err != nil {
		t.Fatalf("resellError(nil) = %v, want nil", err)
	}
}
