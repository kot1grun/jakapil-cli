/*
Copyright © 2026 Kot1gruN
*/
package staticroute

import (
	"fmt"
	"strings"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// staticRouteHeader lists the compact columns shown for the list table format.
var staticRouteHeader = []string{"ID", "Name", "Router ID", "Next Hop", "CIDR", "Status", "Subnet ID", "Tags"}

// staticRouteShowHeader is the two-column header used for the show/results table
// of a single static route: a vertical parameter-per-row layout reads better
// than a wide single-row table.
var staticRouteShowHeader = []string{"Parameter", "Value"}

// render prints the result through the --output flag: json/yaml render the raw
// v1.StaticRoute value, table delegates to the toTable mapper.
func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow prints a single static route with its full attribute set. Long-form
// (json/yaml) output serializes the raw v1.StaticRoute value; the table format
// uses a vertical Parameter/Value layout.
func renderShow(cmd *cobra.Command, r *v1.StaticRoute) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, r)
	}
	if r == nil {
		return output.PrintTable(cmd, staticRouteShowHeader, nil)
	}
	return output.PrintTable(cmd, staticRouteShowHeader, staticRouteShowRows(*r))
}

// toTable maps a single static route or a slice of them to table rows.
func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]v1.StaticRoute); ok {
		rows := make([][]string, 0, len(list))
		for _, r := range list {
			rows = append(rows, staticRouteRow(r))
		}
		return staticRouteHeader, rows, nil
	}
	if single, ok := v.(v1.StaticRoute); ok {
		return staticRouteHeader, [][]string{staticRouteRow(single)}, nil
	}
	if p, ok := v.(*v1.StaticRoute); ok {
		if p == nil {
			return staticRouteHeader, [][]string{}, nil
		}
		return staticRouteHeader, [][]string{staticRouteRow(*p)}, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func staticRouteRow(r v1.StaticRoute) []string {
	return []string{
		r.ID,
		r.Name,
		r.RouterID,
		r.NextHop,
		r.Cidr,
		r.Status,
		r.SubnetID,
		strings.Join(r.Tags, ","),
	}
}

// staticRouteShowRows renders a single static route as vertical Parameter/Value
// rows covering every field the Global Router API exposes.
func staticRouteShowRows(r v1.StaticRoute) [][]string {
	return [][]string{
		{"ID", r.ID},
		{"Name", r.Name},
		{"Router ID", r.RouterID},
		{"Next Hop", r.NextHop},
		{"CIDR", r.Cidr},
		{"Status", r.Status},
		{"Subnet ID", r.SubnetID},
		{"Tags", strings.Join(r.Tags, ",")},
		{"Account ID", r.AccountID},
		{"Project ID", r.ProjectID},
		{"Netops Static Route ID", r.NetopsStaticRouteID},
		{"Created At", r.CreatedAt},
		{"Updated At", r.UpdatedAt},
	}
}
