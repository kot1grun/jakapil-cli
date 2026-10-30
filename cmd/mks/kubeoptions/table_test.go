/*
Copyright © 2026 Kot1gruN
*/

package kubeoptions

import (
	"strings"
	"testing"

	mkskubeoptions "github.com/selectel/mks-go/pkg/v1/kubeoptions"
)

func TestToTable(t *testing.T) {
	header, rows, err := toTable([]*mkskubeoptions.View{
		{KubeVersion: "1.29", Names: []string{"CPUManager", "HPAScaleToZero"}},
	})
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(header, ","); got != "Kube Version,Options" {
		t.Fatalf("header mismatch: %v", got)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	got := strings.Join(rows[0], "|")
	if got != "1.29|CPUManager, HPAScaleToZero" {
		t.Fatalf("unexpected row: %v", rows[0])
	}
}
