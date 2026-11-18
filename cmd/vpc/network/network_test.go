/*
Copyright © 2026 Kot1gruN
*/
package network

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/testutil"
)

// netResource is the network resource served by the fake server for the
// network command tests.
var netResource = testutil.Resource{
	Path:         "networks",
	ItemKey:      "network",
	RequireField: "name",
	List: []map[string]any{
		{"id": "net-1", "name": "nf", "status": "ACTIVE"},
		{"id": "net-2", "name": "second", "status": "DOWN"},
	},
	Get:     map[string]any{"id": "net-1", "name": "nf", "status": "ACTIVE"},
	Updated: map[string]any{"id": "net-1", "name": "renamed", "status": "ACTIVE"},
	Created: func(in map[string]any) map[string]any {
		return map[string]any{"id": "net-new", "name": in["name"], "status": "ACTIVE"}
	},
}

// testCommand builds a bare cobra command carrying the flags the network run
// functions read, with an --output flag and a captured stdout.
func testCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "network"}

	fs := cmd.Flags()
	fs.StringP("output", "o", "yaml", "output format: json or yaml")
	fs.StringP("name", "n", "", "")
	fs.String("description", "", "")
	fs.String("status", "", "")
	fs.String("id", "", "")
	fs.Int("limit", 0, "")
	fs.Bool("admin-state-up", true, "")
	fs.Bool("shared", false, "")

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetContext(context.Background())
	return cmd, &out
}

func TestRunList(t *testing.T) {
	ts := testutil.Neutron(t, netResource)
	cmd, out := testCommand(t)

	if err := runList(cmd, testutil.NewClient(ts)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if !strings.Contains(out.String(), "net-1") || !strings.Contains(out.String(), "nf") {
		t.Fatalf("list output missing network; got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "net-2") {
		t.Fatalf("list output missing second network; got:\n%s", out.String())
	}
}

func TestRunListTable(t *testing.T) {
	ts := testutil.Neutron(t, netResource)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("output", "table"); err != nil {
		t.Fatal(err)
	}

	if err := runList(cmd, testutil.NewClient(ts)); err != nil {
		t.Fatalf("runList: %v", err)
	}
	// The compact table shows only the header columns and the data rows, not
	// the full JSON/YAML record.
	for _, want := range []string{"ID", "Name", "Status", "Shared", "Project ID", "net-1", "nf", "ACTIVE"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("table output missing %q; got:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "admin_state_up") {
		t.Fatalf("table output must not include long-form fields; got:\n%s", out.String())
	}
}

func TestRunShow(t *testing.T) {
	ts := testutil.Neutron(t, netResource)
	cmd, out := testCommand(t)

	if err := runShow(cmd, testutil.NewClient(ts), []string{"net-1"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	if !strings.Contains(out.String(), "net-1") {
		t.Fatalf("show output missing network; got:\n%s", out.String())
	}
}

func TestRunShowTable(t *testing.T) {
	ts := testutil.Neutron(t, netResource)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("output", "table"); err != nil {
		t.Fatal(err)
	}

	if err := runShow(cmd, testutil.NewClient(ts), []string{"net-1"}); err != nil {
		t.Fatalf("runShow: %v", err)
	}
	for _, want := range []string{"Parameter", "Value", "ID", "net-1", "Name", "nf", "Status", "ACTIVE"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("show table output missing %q; got:\n%s", want, out.String())
		}
	}
	if !strings.Contains(out.String(), "╔") || !strings.Contains(out.String(), "║") {
		t.Fatalf("show must render a bordered vertical table:\n%s", out.String())
	}
}

func TestRunCreate(t *testing.T) {
	ts := testutil.Neutron(t, netResource)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "created-net"); err != nil {
		t.Fatal(err)
	}

	if err := runCreate(cmd, testutil.NewClient(ts)); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if !strings.Contains(out.String(), "net-new") || !strings.Contains(out.String(), "created-net") {
		t.Fatalf("create output missing created network; got:\n%s", out.String())
	}
}

func TestRunCreateRequiresName(t *testing.T) {
	ts := testutil.Neutron(t, netResource)
	cmd, _ := testCommand(t)

	if err := runCreate(cmd, testutil.NewClient(ts)); err == nil {
		t.Fatal("expected error when --name is missing")
	}
}

func TestRunUpdate(t *testing.T) {
	ts := testutil.Neutron(t, netResource)
	cmd, out := testCommand(t)
	if err := cmd.Flags().Set("name", "renamed"); err != nil {
		t.Fatal(err)
	}

	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"net-1"}); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if !strings.Contains(out.String(), "renamed") {
		t.Fatalf("update output missing renamed network; got:\n%s", out.String())
	}
}

func TestRunUpdateRequiresOptions(t *testing.T) {
	ts := testutil.Neutron(t, netResource)
	cmd, _ := testCommand(t)

	if err := runUpdate(cmd, testutil.NewClient(ts), []string{"net-1"}); err == nil {
		t.Fatal("expected error when no update option is set")
	}
}

func TestRunDelete(t *testing.T) {
	ts := testutil.Neutron(t, netResource)
	cmd, out := testCommand(t)

	if err := runDelete(cmd, testutil.NewClient(ts), []string{"net-1"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), "network net-1 deleted") {
		t.Fatalf("delete output missing confirmation; got:\n%s", out.String())
	}
}
