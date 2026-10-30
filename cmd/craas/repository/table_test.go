/*
Copyright © 2026 Kot1gruN
*/

package repository

import (
	"strings"
	"testing"
	"time"

	craasrepository "github.com/selectel/craas-go/pkg/v1/repository"
)

func TestToTable(t *testing.T) {
	ts := time.Date(2026, 10, 28, 14, 5, 7, 0, time.UTC)
	header, rows, err := toTable([]*craasrepository.Repository{
		{Name: "app/api", UpdatedAt: ts, Size: 512 * 1024},
	})
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(header, ","); got != "Name,Updated At,Size" {
		t.Fatalf("header mismatch: %v", got)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	got := strings.Join(rows[0], "|")
	if got != "app/api|2026-10-28 14:05:07|512.0 KiB" {
		t.Fatalf("unexpected row: %v", rows[0])
	}
}
