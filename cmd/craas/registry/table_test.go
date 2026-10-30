/*
Copyright © 2026 Kot1gruN
*/

package registry

import (
	"strings"
	"testing"

	craasregistry "github.com/selectel/craas-go/pkg/v1/registry"
)

func TestToTable(t *testing.T) {
	header, rows, err := toTable([]*craasregistry.Registry{
		{ID: "reg-1", Name: "acc/repo", Status: "ACTIVE", Size: 2 * 1024 * 1024, SizeLimit: 2 * 1024 * 1024 * 1024, Used: 0.1},
	})
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(header, ","); got != "ID,Name,Status,Size,Size Limit,Used" {
		t.Fatalf("header mismatch: %v", got)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	got := strings.Join(rows[0], "|")
	for _, c := range []string{"reg-1", "acc/repo", "ACTIVE", "2.0 MiB", "2.0 GiB", "0.1%"} {
		if !strings.Contains(got, c) {
			t.Fatalf("row %q missing %q", got, c)
		}
	}
}
