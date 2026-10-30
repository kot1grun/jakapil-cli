/*
Copyright © 2026 Kot1gruN
*/

package node

import (
	"strings"
	"testing"

	mksnode "github.com/selectel/mks-go/pkg/v1/node"
)

func TestToTable(t *testing.T) {
	header, rows, err := toTable([]*mksnode.View{
		{ID: "n1", Hostname: "worker-1", IP: "10.0.0.5", NodegroupID: "ng-1", OSServerID: "os-1"},
	})
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(header, ","); got != "ID,Hostname,IP,Nodegroup ID,OS Server ID" {
		t.Fatalf("header mismatch: %v", got)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	got := strings.Join(rows[0], "|")
	for _, c := range []string{"n1", "worker-1", "10.0.0.5", "ng-1", "os-1"} {
		if !strings.Contains(got, c) {
			t.Fatalf("row %q missing %q", got, c)
		}
	}
}
