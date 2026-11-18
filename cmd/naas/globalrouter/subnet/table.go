/*
Copyright © 2026 Kot1gruN
*/
package subnet

import (
	"fmt"
	"strings"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// subnetHeader lists the compact columns shown for the list table format.
var subnetHeader = []string{"ID", "Name", "Network ID", "Gateway", "CIDR", "Status", "Tags"}

// subnetShowHeader is the two-column header used for the show/results table of a
// single subnet: a vertical parameter-per-row layout reads better than a wide
// single-row table.
var subnetShowHeader = []string{"Parameter", "Value"}

// render prints the result through the --output flag: json/yaml render the raw
// v1.Subnet value, table delegates to the toTable mapper.
func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow prints a single subnet with its full attribute set. Long-form
// (json/yaml) output serializes the raw v1.Subnet value; the table format uses a
// vertical Parameter/Value layout.
func renderShow(cmd *cobra.Command, s *v1.Subnet) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, s)
	}
	if s == nil {
		return output.PrintTable(cmd, subnetShowHeader, nil)
	}
	return output.PrintTable(cmd, subnetShowHeader, subnetShowRows(*s))
}

// toTable maps a single subnet or a slice of them to table rows.
func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]v1.Subnet); ok {
		rows := make([][]string, 0, len(list))
		for _, s := range list {
			rows = append(rows, subnetRow(s))
		}
		return subnetHeader, rows, nil
	}
	if single, ok := v.(v1.Subnet); ok {
		return subnetHeader, [][]string{subnetRow(single)}, nil
	}
	if p, ok := v.(*v1.Subnet); ok {
		if p == nil {
			return subnetHeader, [][]string{}, nil
		}
		return subnetHeader, [][]string{subnetRow(*p)}, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func subnetRow(s v1.Subnet) []string {
	return []string{
		s.ID,
		s.Name,
		s.NetworkID,
		s.Gateway,
		s.Cidr,
		s.Status,
		strings.Join(s.Tags, ","),
	}
}

// subnetShowRows renders a single subnet as vertical Parameter/Value rows
// covering every field the Global Router API exposes.
func subnetShowRows(s v1.Subnet) [][]string {
	return [][]string{
		{"ID", s.ID},
		{"Name", s.Name},
		{"Network ID", s.NetworkID},
		{"Gateway", s.Gateway},
		{"CIDR", s.Cidr},
		{"Service Addresses", strings.Join(s.ServiceAddresses, ", ")},
		{"OS Subnet ID", s.OsSubnetID},
		{"Tags", strings.Join(s.Tags, ", ")},
		{"Project ID", s.ProjectID},
		{"Status", s.Status},
		{"Created At", s.CreatedAt},
		{"Updated At", s.UpdatedAt},
		{"Account ID", s.AccountID},
		{"Netops Subnet ID", s.NetopsSubnetID},
		{"SV Subnet ID", s.SvSubnetID},
	}
}
