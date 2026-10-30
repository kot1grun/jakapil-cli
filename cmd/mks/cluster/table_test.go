/*
Copyright © 2026 Kot1gruN
*/

package cluster

import (
	"bytes"
	"strings"
	"testing"

	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
	mkstask "github.com/selectel/mks-go/pkg/v1/task"
	"github.com/spf13/cobra"
)

func TestToTableCluster(t *testing.T) {
	header, rows, err := toTable(&mkscluster.GetView{
		ID:          "c1",
		Name:        "prod",
		Status:      "FULLY_OPERATIONAL",
		KubeVersion: "1.29.1",
		Region:      "ru-3",
		ClusterType: "HIGH_AVAILABILITY",
		CNIType:     "CALICO",
	})
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	wantHeader := []string{"ID", "Name", "Status", "Kube Version", "Region", "Cluster Type", "CNI"}
	if strings.Join(header, ",") != strings.Join(wantHeader, ",") {
		t.Fatalf("header mismatch: got %v want %v", header, wantHeader)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	got := strings.Join(rows[0], "|")
	for _, c := range []string{"c1", "prod", "FULLY_OPERATIONAL", "1.29.1", "ru-3", "HIGH_AVAILABILITY", "CALICO"} {
		if !strings.Contains(got, c) {
			t.Fatalf("cluster row %q missing %q", got, c)
		}
	}
}

func TestToTableTask(t *testing.T) {
	header, rows, err := toTable([]*mkstask.View{
		{ID: "task-1", Status: "DONE", Type: "kube-apiserver-phasing", ClusterID: "c1", NodeGroupID: "ng-1"},
	})
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(header, ","); got != "ID,Status,Type,Cluster ID,Nodegroup ID" {
		t.Fatalf("task header mismatch: %v", got)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	got := strings.Join(rows[0], "|")
	for _, c := range []string{"task-1", "DONE", "kube-apiserver-phasing", "c1", "ng-1"} {
		if !strings.Contains(got, c) {
			t.Fatalf("task row %q missing %q", got, c)
		}
	}
}

// TestRenderTable proves the render dispatch produces a bordered table when the
// --output flag is set to table.
func TestRenderTable(t *testing.T) {
	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", "yaml", "")
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := cmd.Flags().Set("output", "table"); err != nil {
		t.Fatal(err)
	}
	if err := render(cmd, []*mkscluster.ListView{{
		ID: "c1", Name: "prod", Status: "ACTIVE", KubeVersion: "1.29.1",
		Region: "ru-3", ClusterType: "BASIC", CNIType: "CALICO",
	}}); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	for _, s := range []string{"ID", "Name", "Status", "prod", "ACTIVE", "1.29.1"} {
		if !strings.Contains(out, s) {
			t.Fatalf("table output missing %q:\n%s", s, out)
		}
	}
	// The renderer must emit a bordered grid, not a JSON/YAML dump.
	if !strings.Contains(out, "┌") || !strings.Contains(out, "│") {
		t.Fatalf("expected bordered table, got:\n%s", out)
	}
	if strings.Contains(out, `"id"`) {
		t.Fatalf("table output must not be JSON:\n%s", out)
	}
}
