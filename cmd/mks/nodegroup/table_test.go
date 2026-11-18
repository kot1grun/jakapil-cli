/*
Copyright © 2026 Kot1gruN
*/

package nodegroup

import (
	"bytes"
	"strings"
	"testing"

	mksnode "github.com/selectel/mks-go/pkg/v1/node"
	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"
)

func TestToTable(t *testing.T) {
	header, rows, err := toTable([]*mksnodegroup.ListView{
		{
			ID: "ng-1", Status: "SHIP_OK", ClusterID: "c1", FlavorID: "f1",
			VolumeGB: 50, VolumeType: "fast.ru-3a", LocalVolume: false,
			NodegroupType: "worker", Nodes: []*mksnode.View{{ID: "n1"}, {ID: "n2"}},
		},
	})
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(header, ","); got != "ID,Status,Cluster ID,Flavor,Volume,Nodes,Type" {
		t.Fatalf("header mismatch: %v", got)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	got := strings.Join(rows[0], "|")
	for _, c := range []string{"ng-1", "SHIP_OK", "c1", "f1", "50GB fast.ru-3a", "2", "worker"} {
		if !strings.Contains(got, c) {
			t.Fatalf("row %q missing %q", got, c)
		}
	}
}

func TestToTableLocalVolume(t *testing.T) {
	_, rows, err := toTable(&mksnodegroup.GetView{LocalVolume: true, Nodes: nil})
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if !strings.Contains(strings.Join(rows[0], "|"), "local") {
		t.Fatalf("expected local volume cell, got %v", rows[0])
	}
}

// TestNodegroupShowRows proves the show table covers the key nodegroup fields in
// a vertical Parameter/Value layout, with deterministic label/taint rendering.
func TestNodegroupShowRows(t *testing.T) {
	rows := nodegroupShowRows(mksnodegroup.GetView{
		ID:                        "ng-1",
		Status:                    "SHIP_OK",
		ClusterID:                 "c1",
		FlavorID:                  "f1",
		VolumeGB:                  50,
		VolumeType:                "fast.ru-3a",
		LocalVolume:               false,
		NodegroupType:             "worker",
		AvailabilityZone:          "ru-3a",
		Nodes:                     []*mksnode.View{{ID: "n1"}, {ID: "n2"}},
		Labels:                    map[string]string{"b": "two", "a": "one"},
		Taints:                    []mksnodegroup.Taint{{Key: "key", Value: "val", Effect: mksnodegroup.NoScheduleEffect}},
		Preemptible:               true,
		InstallNvidiaDevicePlugin: false,
	})
	seen := map[string]string{}
	for _, r := range rows {
		seen[r[0]] = r[1]
	}
	for k, want := range map[string]string{
		"ID": "ng-1", "Status": "SHIP_OK", "Cluster ID": "c1",
		"Flavor": "f1", "Volume": "50GB fast.ru-3a", "Node Count": "2",
		"Nodes":          "n1\nn2",
		"Nodegroup Type": "worker", "Availability Zone": "ru-3a",
		"Preemptible": "yes", "Nvidia Plugin": "no",
		"Labels": "a=one, b=two", "Taints": "key=val:NoSchedule",
	} {
		if got, ok := seen[k]; !ok || got != want {
			t.Fatalf("nodegroup show row %q = %q, want %q (present=%v)", k, got, want, ok)
		}
	}
}

// TestNodegroupShowRowsEmptyLists verifies empty labels/taints and the autoscale
// summary render as "-".
func TestNodegroupShowRowsEmptyLists(t *testing.T) {
	rows := nodegroupShowRows(mksnodegroup.GetView{})
	seen := map[string]string{}
	for _, r := range rows {
		seen[r[0]] = r[1]
	}
	for _, k := range []string{"Labels", "Taints", "Autoscale"} {
		if got, ok := seen[k]; !ok || got != "-" {
			t.Fatalf("nodegroup show row %q = %q, want - (present=%v)", k, got, ok)
		}
	}
}

// TestNodeHostnames verifies the nodes cell enumerates hostnames, one per line,
// falling back to the node ID and dropping nil entries.
func TestNodeHostnames(t *testing.T) {
	cases := []struct {
		name  string
		nodes []*mksnode.View
		want  string
	}{
		{"empty", nil, "-"},
		{"hostnames", []*mksnode.View{{Hostname: "ng-1-0"}, {Hostname: "ng-1-1"}}, "ng-1-0\nng-1-1"},
		{"fallback to id", []*mksnode.View{{ID: "n-id"}, {Hostname: "ng-1-1"}}, "n-id\nng-1-1"},
		{"nil skipped", []*mksnode.View{nil, {Hostname: "ng-1-0"}}, "ng-1-0"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := nodeHostnames(tt.nodes); got != tt.want {
				t.Fatalf("nodeHostnames = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRenderShow proves the show renderer emits a vertical bordered table in
// table mode and raw yaml otherwise.
func TestRenderShow(t *testing.T) {
	g := &mksnodegroup.GetView{ID: "ng-1", ClusterID: "c1"}

	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", "yaml", "")
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	if err := renderShow(cmd, g); err != nil {
		t.Fatalf("renderShow (yaml): %v", err)
	}
	if out := buf.String(); strings.Contains(out, "║") || !strings.Contains(out, "ng-1") {
		t.Fatalf("non-table show output must be raw yaml, got:\n%s", out)
	}

	buf.Reset()
	if err := cmd.Flags().Set("output", "table"); err != nil {
		t.Fatal(err)
	}
	if err := renderShow(cmd, g); err != nil {
		t.Fatalf("renderShow (table): %v", err)
	}
	out := buf.String()
	for _, s := range []string{"Parameter", "Value", "ID", "ng-1", "Cluster ID", "c1"} {
		if !strings.Contains(out, s) {
			t.Fatalf("show table missing %q:\n%s", s, out)
		}
	}
	if !strings.Contains(out, "╔") || !strings.Contains(out, "║") {
		t.Fatalf("show must render a bordered table:\n%s", out)
	}
}
