/*
Copyright © 2026 Kot1gruN
*/

package nodegroup

import (
	"strings"
	"testing"

	mksnode "github.com/selectel/mks-go/pkg/v1/node"
	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
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
