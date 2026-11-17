/*
Copyright © 2026 Kot1gruN
*/
package globalrouter

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// routerHeader lists the compact columns shown for the list table format.
var routerHeader = []string{"ID", "Name", "Status", "Enabled", "Tags"}

// routerShowHeader is the two-column header used for the show/results table of a
// single router: a router has many fields and their values tend to be long, so a
// vertical parameter-per-row layout reads better than a wide single-row table.
var routerShowHeader = []string{"Parameter", "Value"}

// render prints the result through the --output flag: json/yaml render the raw
// v1.Router value (whose fields carry proper json tags), table delegates to the
// toTable mapper.
func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow prints a single router with its full attribute set. Long-form
// (json/yaml) output serializes the raw v1.Router value; the table format uses a
// vertical Parameter/Value layout so every field is visible.
func renderShow(cmd *cobra.Command, r *v1.Router) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, r)
	}
	if r == nil {
		return output.PrintTable(cmd, routerShowHeader, nil)
	}
	return output.PrintTable(cmd, routerShowHeader, routerShowRows(*r))
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

// routerShowRows renders a single router as vertical Parameter/Value rows
// covering every field the Global Router API exposes.
func routerShowRows(r v1.Router) [][]string {
	return [][]string{
		{"ID", r.ID},
		{"Name", r.Name},
		{"Status", r.Status},
		{"Enabled", output.YesNo(r.Enabled)},
		{"Tags", strings.Join(r.Tags, ",")},
		{"Account ID", r.AccountID},
		{"Project ID", r.ProjectID},
		{"Netops Router ID", r.NetopsRouterID},
		{"Prefix Pool ID", r.PrefixPoolID},
		{"VPN ID", strconv.Itoa(r.VpnID)},
		{"Leak UUID", r.LeakUUID},
		{"Created At", r.CreatedAt},
		{"Updated At", r.UpdatedAt},
	}
}
