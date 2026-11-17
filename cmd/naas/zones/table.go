/*
Copyright © 2026 Kot1gruN
*/
package zones

import (
	"fmt"
	"strconv"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// zoneHeader lists the compact columns shown for the zone list table format.
var zoneHeader = []string{"ID", "Name", "Visible Name", "Service", "Enable", "Allow Create", "Allow Update", "Allow Delete", "Groups"}

// render prints the result through the --output flag: json/yaml render the raw
// v1.Zone value (whose fields carry proper json tags), table delegates to the
// toTable mapper.
func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]v1.Zone); ok {
		rows := make([][]string, 0, len(list))
		for _, z := range list {
			rows = append(rows, zoneRow(z))
		}
		return zoneHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func zoneRow(z v1.Zone) []string {
	return []string{
		z.ID,
		z.Name,
		z.VisibleName,
		z.Service,
		output.YesNo(z.Enable),
		output.YesNo(z.AllowCreate),
		output.YesNo(z.AllowUpdate),
		output.YesNo(z.AllowDelete),
		strconv.Itoa(len(z.ZoneGroups)),
	}
}
