/*
Copyright © 2026 Kot1gruN
*/

package kubeversion

import (
	"strings"
	"testing"

	mkskubeversion "github.com/selectel/mks-go/pkg/v1/kubeversion"
)

func TestToTable(t *testing.T) {
	header, rows, err := toTable([]*mkskubeversion.View{
		{Version: "1.29.1", IsDefault: true},
		{Version: "1.28.9", IsDefault: false},
	})
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(header, ","); got != "Version,Default" {
		t.Fatalf("header mismatch: %v", got)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if r := strings.Join(rows[0], "|"); r != "1.29.1|yes" {
		t.Fatalf("unexpected first row: %v", rows[0])
	}
	if r := strings.Join(rows[1], "|"); r != "1.28.9|no" {
		t.Fatalf("unexpected second row: %v", rows[1])
	}
}

func TestToTableRejectsNonSlice(t *testing.T) {
	if _, _, err := toTable("nope"); err == nil {
		t.Fatal("expected error for non-slice input")
	}
}
