/*
Copyright © 2026 Kot1gruN
*/

package token

import (
	"strings"
	"testing"
	"time"

	craastoken "github.com/selectel/craas-go/pkg/v1/token"
	tokenv2 "github.com/selectel/craas-go/pkg/v2/token"
)

func TestToTableV2(t *testing.T) {
	exp := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	data := &tokenv2.TokensV2{Tokens: []tokenv2.TokenV2{{
		ID: "tok-1", Name: "ci", Status: "active",
		Scope:      tokenv2.Scope{ModeRW: true, RegistryIDs: []string{"reg-1", "reg-2"}},
		Expiration: tokenv2.Expiration{IsSet: true, ExpiresAt: exp},
	}}}
	header, rows, err := toTable(data)
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(header, ","); got != "ID,Name,Status,Mode,Registries,Expires At" {
		t.Fatalf("header mismatch: %v", got)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	got := strings.Join(rows[0], "|")
	for _, c := range []string{"tok-1", "ci", "active", "rw", "reg-1,reg-2", "2027-01-01 00:00:00"} {
		if !strings.Contains(got, c) {
			t.Fatalf("row %q missing %q", got, c)
		}
	}
}

func TestToTableV2EmptyRegistries(t *testing.T) {
	data := &tokenv2.TokensV2{Tokens: []tokenv2.TokenV2{{ID: "tok-1"}}}
	_, rows, err := toTable(data)
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(rows[0], "|"); !strings.Contains(got, "ro") || !strings.Contains(got, "-") {
		t.Fatalf("expected read-only mode and dash registries, got %v", rows[0])
	}
}

func TestToTableV1(t *testing.T) {
	data := &craastoken.Token{Token: "secret", ExpiresAt: 0}
	header, rows, err := toTable(data)
	if err != nil {
		t.Fatalf("toTable: %v", err)
	}
	if got := strings.Join(header, ","); got != "Token,Expires At" {
		t.Fatalf("header mismatch: %v", got)
	}
	got := strings.Join(rows[0], "|")
	if !strings.Contains(got, "secret") || !strings.Contains(got, "-") {
		t.Fatalf("unexpected v1 row: %v", rows[0])
	}
}
