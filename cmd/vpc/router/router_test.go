/*
Copyright © 2026 Kot1gruN
*/
package router

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/spf13/cobra"
)

// routerListBody mirrors the JSON served by the gophercloud
// layer3/routers/testing package for a routers.List call.
const routerListBody = `{
    "routers": [
        {
            "status": "ACTIVE",
            "external_gateway_info": null,
            "name": "second_routers",
            "admin_state_up": true,
            "distributed": false,
            "id": "7177abc4-5ae9-4bb7-b0d4-89e94a4abf3b"
        },
        {
            "status": "ACTIVE",
            "external_gateway_info": {
                "network_id": "3c5bcddd-6af9-4e6b-9c3e-c153e521cab8"
            },
            "name": "router1",
            "admin_state_up": true,
            "distributed": true,
            "id": "a9254bdb-2613-4a13-ac4c-adc581fba50d"
        }
    ]
}`

// testCommand builds a bare "router" command with the flags shared by all
// subcommands so that the run* helpers can be exercised directly.
func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "router"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.String("description", "", "")
	fs.String("status", "", "")
	fs.String("id", "", "")
	fs.Int("limit", 0, "")
	fs.Bool("admin-state-up", true, "")
	fs.Bool("distributed", false, "")
	fs.String("gateway-network-id", "", "")
	fs.Bool("enable-snat", true, "")
	fs.Bool("disable-snat", false, "")
	fs.String("subnet-id", "", "")
	fs.String("port-id", "", "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func verifyRequest(t *testing.T, r *http.Request, method string) {
	t.Helper()
	th.TestMethod(t, r, method)
	th.TestHeader(t, r, "X-Auth-Token", common.TokenID)
}

func TestRunList(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/v2.0/routers", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "GET")
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, routerListBody)
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, common.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"7177abc4-5ae9-4bb7-b0d4-89e94a4abf3b", "second_routers", "router1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}
func TestRunCreate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/v2.0/routers", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "POST")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `{
   "router":{
      "name":"foo_router",
      "admin_state_up":false,
      "external_gateway_info":{
         "network_id":"8ca37218-28ff-41cb-9b10-039601ea7e6b"
      }
   }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
    "router": {
        "status": "ACTIVE",
        "external_gateway_info": {
            "network_id": "8ca37218-28ff-41cb-9b10-039601ea7e6b"
        },
        "name": "foo_router",
        "admin_state_up": false,
        "distributed": false,
        "id": "8604a0de-7f6b-409a-a47c-a1cc7bc77b2e"
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "foo_router"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("admin-state-up", "false"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("gateway-network-id", "8ca37218-28ff-41cb-9b10-039601ea7e6b"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, common.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	for _, want := range []string{"foo_router", "8604a0de-7f6b-409a-a47c-a1cc7bc77b2e"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("create output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunUpdate(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/v2.0/routers/4e8e5957-649f-477b-9e5b-f1f75b21c03c", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "PUT")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `{
    "router": {
        "name": "new_name"
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "router": {
        "status": "ACTIVE",
        "name": "new_name",
        "admin_state_up": true,
        "distributed": false,
        "id": "4e8e5957-649f-477b-9e5b-f1f75b21c03c"
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "new_name"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, common.ServiceClient(fakeServer), []string{"4e8e5957-649f-477b-9e5b-f1f75b21c03c"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "new_name") {
		t.Fatalf("update output missing %q; got:\n%s", "new_name", out.String())
	}
}

func TestRunUpdateRequiresOptions(t *testing.T) {
	cmd, _ := testCommand(t)
	err := runUpdate(cmd, nil, []string{"r-1"})
	if err == nil {
		t.Fatal("runUpdate succeeded without any update option; want an error")
	}
	if !strings.Contains(err.Error(), "at least one update option") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDelete(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/v2.0/routers/8604a0de-7f6b-409a-a47c-a1cc7bc77b2e", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	cmd, out := testCommand(t)
	if err := runDelete(cmd, common.ServiceClient(fakeServer), []string{"8604a0de-7f6b-409a-a47c-a1cc7bc77b2e"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if want := "router 8604a0de-7f6b-409a-a47c-a1cc7bc77b2e deleted"; !strings.Contains(out.String(), want) {
		t.Fatalf("delete output missing %q; got:\n%s", want, out.String())
	}
}

// compile-time guard that the helpers used above map to the routers API.
func TestRunPortAdd(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	const routerID = "4e8e5957-649f-477b-9e5b-f1f75b21c03c"
	fakeServer.Mux.HandleFunc("/v2.0/routers/"+routerID+"/add_router_interface", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "PUT")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `{
    "subnet_id": "a2f1f29d-571b-4533-907f-5803ab96ead1"
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "subnet_id": "0d32a837-8069-4ec3-84c4-3eef3e10b188",
    "tenant_id": "017d8de156df4177889f31a9bd6edc00",
    "port_id": "3f990102-4485-4df1-97a0-2c35bdb85b31",
    "id": "9a83fa11-8da5-436e-9afe-3d3ac5ce7770"
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("subnet-id", "a2f1f29d-571b-4533-907f-5803ab96ead1"); err != nil {
		t.Fatal(err)
	}

	if err := runPortAdd(cmd, common.ServiceClient(fakeServer), []string{routerID}); err != nil {
		t.Fatalf("runPortAdd: %v", err)
	}
	if want := "attached interface to router " + routerID; !strings.Contains(out.String(), want) {
		t.Fatalf("port add output missing %q; got:\n%s", want, out.String())
	}
}

func TestRunPortAddRequiresOneOf(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runPortAdd(cmd, nil, []string{"r-1"}); err == nil {
		t.Fatal("runPortAdd succeeded without subnet/port; want an error")
	}
	if err := cmd.Flags().Set("subnet-id", "sub-1"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("port-id", "port-1"); err != nil {
		t.Fatal(err)
	}
	if err := runPortAdd(cmd, nil, []string{"r-1"}); err == nil {
		t.Fatal("runPortAdd succeeded with both subnet and port; want an error")
	}
}

func TestRunPortDelete(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	const routerID = "4e8e5957-649f-477b-9e5b-f1f75b21c03c"
	fakeServer.Mux.HandleFunc("/v2.0/routers/"+routerID+"/remove_router_interface", func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "PUT")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `{
			"subnet_id": "a2f1f29d-571b-4533-907f-5803ab96ead1"
		}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"subnet_id": "0d32a837-8069-4ec3-84c4-3eef3e10b188",
			"tenant_id": "017d8de156df4177889f31a9bd6edc00",
			"port_id": "3f990102-4485-4df1-97a0-2c35bdb85b31",
			"id": "9a83fa11-8da5-436e-9afe-3d3ac5ce7770"
		}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("subnet-id", "a2f1f29d-571b-4533-907f-5803ab96ead1"); err != nil {
		t.Fatal(err)
	}

	if err := runPortDelete(cmd, common.ServiceClient(fakeServer), []string{routerID}); err != nil {
		t.Fatalf("runPortDelete: %v", err)
	}
	if want := "detached interface from router " + routerID; !strings.Contains(out.String(), want) {
		t.Fatalf("port delete output missing %q; got:\n%s", want, out.String())
	}
}

func TestRunPortDeleteRequiresOpts(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runPortDelete(cmd, nil, []string{"r-1"}); err == nil {
		t.Fatal("runPortDelete succeeded without subnet/port; want an error")
	}
}

// compile-time guard that the helpers used above map to the routers API.
var _ = routers.ListOpts{}
var _ = routers.ListOpts{}

func TestRunGatewayAdd(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	const routerID = "4e8e5957-649f-477b-9e5b-f1f75b21c03c"
	fakeServer.Mux.HandleFunc("/v2.0/routers/"+routerID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "PUT")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `{
    "router": {
        "external_gateway_info": {
            "network_id": "8ca37218-28ff-41cb-9b10-039601ea7e6b",
            "enable_snat": false
        }
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "router": {
        "id": "`+routerID+`",
        "status": "ACTIVE",
        "name": "router1",
        "admin_state_up": true,
        "distributed": false,
        "external_gateway_info": {
            "network_id": "8ca37218-28ff-41cb-9b10-039601ea7e6b",
            "enable_snat": false
        }
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("gateway-network-id", "8ca37218-28ff-41cb-9b10-039601ea7e6b"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("enable-snat", "false"); err != nil {
		t.Fatal(err)
	}

	if err := runGatewayAdd(cmd, common.ServiceClient(fakeServer), []string{routerID}); err != nil {
		t.Fatalf("runGatewayAdd: %v", err)
	}
	if !strings.Contains(out.String(), routerID) {
		t.Fatalf("gateway add output missing updated router; got:\n%s", out.String())
	}
}

func TestRunGatewayAddRequiresNetwork(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := runGatewayAdd(cmd, nil, []string{"r-1"}); err == nil {
		t.Fatal("runGatewayAdd succeeded without gateway-network-id; want an error")
	}
}

func TestRunGatewayDelete(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	const routerID = "4e8e5957-649f-477b-9e5b-f1f75b21c03c"
	fakeServer.Mux.HandleFunc("/v2.0/routers/"+routerID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "PUT")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `{
    "router": {
        "external_gateway_info": null
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "router": {
        "id": "`+routerID+`",
        "status": "ACTIVE",
        "name": "router1",
        "admin_state_up": true,
        "distributed": false,
        "external_gateway_info": null
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := runGatewayDelete(cmd, common.ServiceClient(fakeServer), []string{routerID}); err != nil {
		t.Fatalf("runGatewayDelete: %v", err)
	}
	if want := "removed external gateway from router " + routerID; !strings.Contains(out.String(), want) {
		t.Fatalf("gateway delete output missing %q; got:\n%s", want, out.String())
	}
}

// compile-time guard that the helpers used above map to the routers API.
var _ = routers.ListOpts{}

func TestRunUpdateEnableSNAT(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	const routerID = "4e8e5957-649f-477b-9e5b-f1f75b21c03c"
	fakeServer.Mux.HandleFunc("/v2.0/routers/"+routerID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "PUT")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `{
    "router": {
        "external_gateway_info": {
            "network_id": "8ca37218-28ff-41cb-9b10-039601ea7e6b",
            "enable_snat": true
        }
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "router": {
        "id": "`+routerID+`",
        "name": "router1",
        "status": "ACTIVE",
        "admin_state_up": true,
        "external_gateway_info": {
            "network_id": "8ca37218-28ff-41cb-9b10-039601ea7e6b",
            "enable_snat": true
        }
    }
}`)
	})

	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("gateway-network-id", "8ca37218-28ff-41cb-9b10-039601ea7e6b"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("enable-snat", "true"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, common.ServiceClient(fakeServer), []string{routerID}); err != nil {
		t.Fatalf("runUpdate with --enable-snat: %v", err)
	}
}

func TestRunUpdateDisableSNAT(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	const routerID = "4e8e5957-649f-477b-9e5b-f1f75b21c03c"
	fakeServer.Mux.HandleFunc("/v2.0/routers/"+routerID, func(w http.ResponseWriter, r *http.Request) {
		verifyRequest(t, r, "PUT")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestJSONRequest(t, r, `{
    "router": {
        "external_gateway_info": {
            "network_id": "8ca37218-28ff-41cb-9b10-039601ea7e6b",
            "enable_snat": false
        }
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
    "router": {
        "id": "`+routerID+`",
        "name": "router1",
        "status": "ACTIVE",
        "admin_state_up": true,
        "external_gateway_info": {
            "network_id": "8ca37218-28ff-41cb-9b10-039601ea7e6b",
            "enable_snat": false
        }
    }
}`)
	})

	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("gateway-network-id", "8ca37218-28ff-41cb-9b10-039601ea7e6b"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("disable-snat", "true"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, common.ServiceClient(fakeServer), []string{routerID}); err != nil {
		t.Fatalf("runUpdate with --disable-snat: %v", err)
	}
}

func TestRunUpdateSnatRequiresGateway(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("enable-snat", "true"); err != nil {
		t.Fatal(err)
	}
	if err := runUpdate(cmd, nil, []string{"r-1"}); err == nil {
		t.Fatal("runUpdate with --enable-snat but no --gateway-network-id succeeded; want an error")
	}
}

func TestRunUpdateSnatMutuallyExclusive(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("enable-snat", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("disable-snat", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("gateway-network-id", "net-1"); err != nil {
		t.Fatal(err)
	}
	if err := runUpdate(cmd, nil, []string{"r-1"}); err == nil {
		t.Fatal("runUpdate with both --enable-snat and --disable-snat succeeded; want an error")
	}
}

// compile-time guard that the helpers used above map to the routers API.
var _ = routers.ListOpts{}
