/*
Copyright © 2026 Kot1gruN
*/
package port

import (
	"bytes"
	"context"
	"strings"
	"testing"

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
	Get:     nil,
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
