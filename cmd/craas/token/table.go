/*
Copyright © 2026 Kot1gruN
*/

package token

import (
	"fmt"
	"strings"
	"time"

	craastoken "github.com/selectel/craas-go/pkg/v1/token"
	tokenv2 "github.com/selectel/craas-go/pkg/v2/token"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// tokenV2Header lists the compact columns shown for the V2 token table.
var tokenV2Header = []string{"ID", "Name", "Status", "Mode", "Registries", "Expires At"}

// tokenV1Header lists the compact columns shown for the V1 token table.
var tokenV1Header = []string{"Token", "Expires At"}

const timeLayout = "2006-01-02 15:04:05"

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	// The V2 list returns a *TokensV2 wrapper (not a plain slice).
	if data, ok := v.(*tokenv2.TokensV2); ok {
		rows := make([][]string, 0, len(data.Tokens))
		for i := range data.Tokens {
			rows = append(rows, tokenV2Row(&data.Tokens[i]))
		}
		return tokenV2Header, rows, nil
	}
	if rows := output.RowsOf(v, tokenV2Row); rows != nil {
		return tokenV2Header, rows, nil
	}
	if rows := output.RowsOf(v, tokenV1Row); rows != nil {
		return tokenV1Header, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func tokenV2Row(t *tokenv2.TokenV2) []string {
	mode := "ro"
	if t.Scope.ModeRW {
		mode = "rw"
	}
	regs := strings.Join(t.Scope.RegistryIDs, ",")
	if regs == "" {
		regs = "-"
	}
	expires := "-"
	if t.Expiration.IsSet {
		expires = t.Expiration.ExpiresAt.Format(timeLayout)
	}
	return []string{t.ID, t.Name, t.Status, mode, regs, expires}
}

func tokenV1Row(t *craastoken.Token) []string {
	expires := "-"
	if t.ExpiresAt > 0 {
		expires = time.Unix(t.ExpiresAt, 0).Format(timeLayout)
	}
	return []string{t.Token, expires}
}
