/*
Copyright © 2026 Kot1gruN
*/
package port

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/testutil"
)

// portResource is the port resource served by the fake server for the
// port command tests. Ports have no "show" subcommand, so Get is nil.
var portResource = testutil.Resource{
	Path:         "ports",
	ItemKey:      "port",
	RequireField: "network_id",
	List: []map[string]any{
		{"id": "port-1", "name": "pf", "network_id": "net-1", "status": "ACTIVE"},
		{"id": "port-2", "name": "second", "network_id": "net-1", "status": "DOWN"},
	},
	Get:     map[string]any{"id": "port-1", "name": "pf", "network_id": "net-1", "status": "ACTIVE", "allowed_address_pairs": []map[string]any{{"ip_address": "10.0.0.2"}}},
	Updated: map[string]any{"id": "port-1", "name": "renamed", "network_id": "net-1", "status": "ACTIVE"},
	Created: func(in map[string]any) map[string]any {
		return map[string]any{"id": "port-new", "name": "pf", "network_id": in["network_id"], "status": "DOWN"}
	},
}

func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "port"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.String("description", "", "")
	fs.String("network-id", "", "")
	fs.String("status", "", "")
	fs.String("mac-address", "", "")
	fs.String("device-id", "", "")
	fs.String("device-owner", "", "")
	fs.String("id", "", "")
	fs.Int("limit", 0, "")
	fs.Bool("admin-state-up", true, "")
	fs.Bool("port-security-enabled", true, "")
	fs.StringSlice("allowed-address-pair", nil, "")
	fs.Bool("append-aap", false, "")
	fs.StringArray("fixed-ip", nil, "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunList(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, out := testCommand(t)

	if err := runList(cmd, testutil.NewClient(ts)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"port-1", "pf", "port-2"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreate(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("network-id", "net-1"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("name", "created-port"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, testutil.NewClient(ts)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "port-new") {
		t.Fatalf("create output missing created port; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresNetwork(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, _ := testCommand(t)

	if err := runCreate(cmd, testutil.NewClient(ts)); err == nil {
		t.Fatal("expected error when --network-id is missing")
	}
}

func TestRunUpdate(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "renamed"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"port-1"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Fatalf("update output missing renamed port; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOptions(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, _ := testCommand(t)

	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"port-1"}); err == nil {
		t.Fatal("expected error when no update option is set")
	}
}

func TestRunUpdateAllowedAddressPairs(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("allowed-address-pair", "10.0.0.5,aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"port-1"}); err != nil {
		t.Fatalf("runUpdate with --allowed-address-pair: %v", err)
	}
}

func TestRunUpdateDisablesPortSecurity(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("port-security-enabled", "false"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"port-1"}); err != nil {
		t.Fatalf("runUpdate with --port-security-enabled=false: %v", err)
	}
}

func TestRunUpdateAppendAAP(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("allowed-address-pair", "10.0.0.5,aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("append-aap", "true"); err != nil {
		t.Fatal(err)
	}

	// runUpdate fetches the port's current pairs (10.0.0.2 from the fake Get)
	// and appends the new pair instead of replacing them. Without the fetch the
	// fake server has no handler for GET /ports/port-1 beyond the list.
	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"port-1"}); err != nil {
		t.Fatalf("runUpdate with --append-aap: %v", err)
	}
}

func TestRunUpdateAppendAAPRequiresPairs(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("append-aap", "true"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"port-1"}); err == nil {
		t.Fatal("expected error when --append-aap is used without --allowed-address-pair")
	}
}

func TestRunDelete(t *testing.T) {
	ts := testutil.Neutron(t, portResource)
	cmd, out := testCommand(t)

	if err := runDelete(cmd, testutil.NewClient(ts), []string{"port-1"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "port port-1 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
func TestRunCreateFixedIP(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", common.TokenID)
		th.TestJSONRequest(t, r, `{
    "port": {
        "network_id": "net-1",
        "name": "created-port",
        "fixed_ips": [
            {"subnet_id": "sub-1", "ip_address": "10.0.0.10"}
        ]
    }
}`)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
    "port": {
        "id": "port-new",
        "name": "created-port",
        "network_id": "net-1",
        "status": "DOWN",
        "fixed_ips": [
            {"subnet_id": "sub-1", "ip_address": "10.0.0.10"}
        ]
    }
}`)
	})

	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("network-id", "net-1"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("name", "created-port"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("fixed-ip", "subnet=sub-1,ip-address=10.0.0.10"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, common.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runCreate with --fixed-ip: %v", err)
	}
	if !strings.Contains(out.String(), "port-new") {
		t.Fatalf("create output missing created port; got:\n%s", out.String())
	}
}

func TestFixedIPsParsing(t *testing.T) {
	cmd, _ := testCommand(t)
	if err := cmd.Flags().Set("fixed-ip", "subnet=sub-1,ip-address=10.0.0.10"); err != nil {
		t.Fatal(err)
	}
	ips, err := fixedIPs(cmd)
	if err != nil {
		t.Fatalf("fixedIPs: %v", err)
	}
	if len(ips) != 1 || ips[0].SubnetID != "sub-1" || ips[0].IPAddress != "10.0.0.10" {
		t.Fatalf("unexpected fixedIPs: %+v", ips)
	}
}

func TestFixedIPsParsingErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		val  string
	}{
		{"missing value", "subnet"},
		{"unknown key", "foo=bar"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd, _ := testCommand(t)
			if err := cmd.Flags().Set("fixed-ip", tc.val); err != nil {
				t.Fatal(err)
			}
			if _, err := fixedIPs(cmd); err == nil {
				t.Fatalf("fixedIPs(%q) succeeded; want an error", tc.val)
			}
		})
	}
}
