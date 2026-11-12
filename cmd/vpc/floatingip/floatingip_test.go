/*
Copyright © 2026 Kot1gruN
*/
package floatingip

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/common"
	ftesting "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips/testing"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/testutil"
)

// fipResource is the floating IP resource served by the fake server for the
// floating-ip command tests. Floating IPs are named by ID, matching real
// Neutron floating IP resources (they have no user-visible name).
var fipResource = testutil.Resource{
	Path:         "floatingips",
	ItemKey:      "floatingip",
	RequireField: "floating_network_id",
	List: []map[string]any{
		{
			"id":                  "fip-1",
			"floating_ip_address": "203.0.113.10",
			"fixed_ip_address":    "192.0.2.5",
			"port_id":             "port-1",
			"floating_network_id": "net-ext-1",
			"status":              "ACTIVE",
			"description":         "web frontend",
		},
		{
			"id":                  "fip-2",
			"floating_ip_address": "203.0.113.11",
			"status":              "DOWN",
			"floating_network_id": "net-ext-1",
		},
	},
	Get: map[string]any{
		"id":                  "fip-1",
		"floating_ip_address": "203.0.113.10",
		"fixed_ip_address":    "192.0.2.5",
		"port_id":             "port-1",
		"floating_network_id": "net-ext-1",
		"status":              "ACTIVE",
	},
	Updated: map[string]any{
		"id":                  "fip-1",
		"floating_ip_address": "203.0.113.10",
		"port_id":             "port-2",
		"floating_network_id": "net-ext-1",
		"status":              "ACTIVE",
	},
	Created: func(in map[string]any) map[string]any {
		return map[string]any{
			"id":                  "fip-new",
			"floating_ip_address": "203.0.113.20",
			"floating_network_id": in["floating_network_id"],
			"status":              "DOWN",
		}
	},
}

func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "floating-ip"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.String("id", "", "")
	fs.String("description", "", "")
	fs.String("floating-network-id", "", "")
	fs.String("port-id", "", "")
	fs.String("fixed-ip-address", "", "")
	fs.String("floating-ip-address", "", "")
	fs.String("status", "", "")
	fs.Int("limit", 0, "")
	fs.String("subnet-id", "", "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunList(t *testing.T) {
	ts := testutil.Neutron(t, fipResource)
	cmd, out := testCommand(t)

	if err := runList(cmd, testutil.NewClient(ts)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	for _, want := range []string{"fip-1", "203.0.113.10", "fip-2", "203.0.113.11"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunShow(t *testing.T) {
	ts := testutil.Neutron(t, fipResource)
	cmd, out := testCommand(t)

	if err := runShow(cmd, testutil.NewClient(ts), []string{"fip-1"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"fip-1", "203.0.113.10", "192.0.2.5", "port-1"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show output missing %q; got:\n%s", want, out.String())
		}
	}
}

func TestRunCreate(t *testing.T) {
	ts := testutil.Neutron(t, fipResource)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("floating-network-id", "net-ext-1"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("description", "db"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, testutil.NewClient(ts)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "fip-new") {
		t.Fatalf("create output missing created IP; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresNetwork(t *testing.T) {
	ts := testutil.Neutron(t, fipResource)
	cmd, _ := testCommand(t)

	if err := runCreate(cmd, testutil.NewClient(ts)); err == nil {
		t.Fatal("expected error when --floating-network-id is missing")
	}
}

func TestRunUpdate(t *testing.T) {
	ts := testutil.Neutron(t, fipResource)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("port-id", "port-2"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"fip-1"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "port-2") {
		t.Fatalf("update output missing new port; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOption(t *testing.T) {
	ts := testutil.Neutron(t, fipResource)
	cmd, _ := testCommand(t)

	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"fip-1"}); err == nil {
		t.Fatal("expected error when no update option is provided")
	}
}

func TestRunDelete(t *testing.T) {
	ts := testutil.Neutron(t, fipResource)
	cmd, out := testCommand(t)

	if err := runDelete(cmd, testutil.NewClient(ts), []string{"fip-1"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "floating IP fip-1 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}

// TestRunListUsesFloatingIPsTesting drives runList against a raw handler that
// serves the fixtures shipped in gophercloud's own floatingips/testing package
// (the exact ListResponse used by the library's tests). It guards against the
// CLI silently diverging from the real floating IP JSON wire format.
func TestRunListUsesFloatingIPsTesting(t *testing.T) {
	fakeServer := th.SetupHTTP()
	t.Cleanup(fakeServer.Teardown)

	fakeServer.Mux.HandleFunc("/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", common.TokenID)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, ftesting.ListResponse)
	})

	cmd, out := testCommand(t)
	if err := runList(cmd, common.ServiceClient(fakeServer)); err != nil {
		t.Fatalf("runList with floatingips/testing fixtures: %v", err)
	}
	for _, want := range []string{"192.0.0.4", "10.0.0.3"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("list output missing %q; got:\n%s", want, out.String())
		}
	}
}
