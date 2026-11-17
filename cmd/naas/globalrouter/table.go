/*
Copyright © 2026 Kot1gruN
*/
package globalrouter

import (
	"fmt"
	"strings"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// routerHeader lists the compact columns shown for the table format.
var routerHeader = []string{"ID", "Name", "Status", "Enabled", "Tags"}

// render prints the result through the --output flag: json/yaml render the raw
// v1.Router value (whose fields carry proper json tags), table delegates to the
// toTable mapper.
func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// toTable maps a single router or a slice of them to table rows.
func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]v1.Router); ok {
		rows := make([][]string, 0, len(list))
		for _, r := range list {
			rows = append(rows, routerRow(r))
		}
		return routerHeader, rows, nil
	}
	if single, ok := v.(v1.Router); ok {
		return routerHeader, [][]string{routerRow(single)}, nil
	}
	if p, ok := v.(*v1.Router); ok {
		if p == nil {
			return routerHeader, [][]string{}, nil
		}
		return routerHeader, [][]string{routerRow(*p)}, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func routerRow(r v1.Router) []string {
	return []string{
		r.ID,
		r.Name,
		r.Status,
		output.YesNo(r.Enabled),
		strings.Join(r.Tags, ","),
	}
}
