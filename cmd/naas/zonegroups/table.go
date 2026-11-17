/*
Copyright © 2026 Kot1gruN
*/
package zonegroups

import (
	"fmt"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// zoneGroupHeader lists the compact columns shown for the zone group list table
// format.
var zoneGroupHeader = []string{"ID", "Name", "Description"}

// render prints the result through the --output flag: json/yaml render the raw
// v1.ZoneGroup value (whose fields carry proper json tags), table delegates to
// the toTable mapper.
func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]v1.ZoneGroup); ok {
		rows := make([][]string, 0, len(list))
		for _, g := range list {
			rows = append(rows, zoneGroupRow(g))
		}
		return zoneGroupHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func zoneGroupRow(g v1.ZoneGroup) []string {
	return []string{
		g.ID,
		g.Name,
		g.Description,
	}
}
