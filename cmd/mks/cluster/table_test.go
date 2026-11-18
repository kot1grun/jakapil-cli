/*
Copyright © 2026 Kot1gruN
*/

package cluster

import (
	"bytes"
	"strings"
	"testing"
	"time"

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
	if !strings.Contains(out, "╔") || !strings.Contains(out, "║") {
		t.Fatalf("expected bordered table, got:\n%s", out)
	}
	if strings.Contains(out, `"id"`) {
		t.Fatalf("table output must not be JSON:\n%s", out)
	}
}

// TestClusterShowRows proves the show table covers the key cluster fields in a
// vertical Parameter/Value layout.
func TestClusterShowRows(t *testing.T) {
	rows := clusterShowRows(mkscluster.GetView{
		ID:               "c1",
		Name:             "prod",
		Status:           "ACTIVE",
		Region:           "ru-3",
		KubeVersion:      "1.29.1",
		ClusterType:      "HIGH_AVAILABILITY",
		CNIType:          "CILIUM",
		ProjectID:        "p1",
		NetworkID:        "net-1",
		SubnetID:         "sub-1",
		KubeAPIIP:        "10.1.0.1",
		PrivateKubeAPI:   true,
		EnableAutorepair: true,
	})
	seen := map[string]string{}
	for _, r := range rows {
		seen[r[0]] = r[1]
	}
	for k, want := range map[string]string{
		"ID": "c1", "Name": "prod", "Status": "ACTIVE", "Region": "ru-3",
		"Kube Version": "1.29.1", "Cluster Type": "HIGH_AVAILABILITY", "CNI": "CILIUM",
		"Project ID": "p1", "Network ID": "net-1", "Subnet ID": "sub-1",
		"Kube API IP": "10.1.0.1", "Private Kube API": "yes", "Autorepair": "yes",
	} {
		if got, ok := seen[k]; !ok || got != want {
			t.Fatalf("cluster show row %q = %q, want %q (present=%v)", k, got, want, ok)
		}
	}
}

// TestRenderShow proves the show renderer switches between a vertical table
// (table) and raw serialization (json/yaml).
func TestRenderShow(t *testing.T) {
	cl := &mkscluster.GetView{ID: "c1", Name: "prod", Region: "ru-3"}

	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", "yaml", "")
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	// Non-table format falls through to raw Print (json/yaml).
	if err := renderShow(cmd, cl); err != nil {
		t.Fatalf("renderShow (yaml): %v", err)
	}
	if out := buf.String(); strings.Contains(out, "║") || !strings.Contains(out, "prod") {
		t.Fatalf("non-table show output must be raw yaml, got:\n%s", out)
	}

	// Table format uses a vertical Parameter/Value layout.
	buf.Reset()
	if err := cmd.Flags().Set("output", "table"); err != nil {
		t.Fatal(err)
	}
	if err := renderShow(cmd, cl); err != nil {
		t.Fatalf("renderShow (table): %v", err)
	}
	out := buf.String()
	for _, s := range []string{"Parameter", "Value", "ID", "Name", "c1", "prod"} {
		if !strings.Contains(out, s) {
			t.Fatalf("show table missing %q:\n%s", s, out)
		}
	}
	if !strings.Contains(out, "╔") || !strings.Contains(out, "║") {
		t.Fatalf("show must render a bordered table:\n%s", out)
	}
}

func TestMaintenanceWindow(t *testing.T) {
	tests := []struct {
		start, end, want string
	}{
		{"", "", "-"},
		{"01:00:00", "", "01:00:00"},
		{"", "03:00:00", "03:00:00"},
		{"01:00:00", "03:00:00", "01:00:00 - 03:00:00"},
	}
	for _, tt := range tests {
		if got := maintenanceWindow(tt.start, tt.end); got != tt.want {
			t.Fatalf("maintenanceWindow(%q, %q) = %q, want %q", tt.start, tt.end, got, tt.want)
		}
	}
}

func TestFormatTime(t *testing.T) {
	if got := formatTime(nil); got != "-" {
		t.Fatalf("formatTime(nil) = %q, want -", got)
	}
	ts := time.Date(2025, 12, 8, 18, 57, 57, 0, time.UTC)
	if got := formatTime(&ts); got != "2025-12-08T18:57:57Z" {
		t.Fatalf("formatTime = %q", got)
	}
}
