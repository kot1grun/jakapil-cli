/*
Copyright © 2026 Kot1gruN
*/
package network

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// networkHeader lists the compact columns shown for the list table format.
var networkHeader = []string{"ID", "Name", "Router ID", "Zone ID", "Status", "VLAN", "Os Network ID", "Tags"}

// networkShowHeader is the two-column header used for the show/results table of
// a single network: a vertical parameter-per-row layout reads better than a wide
// single-row table.
var networkShowHeader = []string{"Parameter", "Value"}

// render prints the result through the --output flag: json/yaml render the raw
// v1.Network value, table delegates to the toTable mapper.
func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow prints a single network with its full attribute set. Long-form
// (json/yaml) output serializes the raw v1.Network value; the table format uses
// a vertical Parameter/Value layout.
func renderShow(cmd *cobra.Command, n *v1.Network) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, n)
	}
	if n == nil {
		return output.PrintTable(cmd, networkShowHeader, nil)
	}
	return output.PrintTable(cmd, networkShowHeader, networkShowRows(*n))
}

// toTable maps a single network or a slice of them to table rows.
func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]v1.Network); ok {
		rows := make([][]string, 0, len(list))
		for _, n := range list {
			rows = append(rows, networkRow(n))
		}
		return networkHeader, rows, nil
	}
	if single, ok := v.(v1.Network); ok {
		return networkHeader, [][]string{networkRow(single)}, nil
	}
	if p, ok := v.(*v1.Network); ok {
		if p == nil {
			return networkHeader, [][]string{}, nil
		}
		return networkHeader, [][]string{networkRow(*p)}, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func networkRow(n v1.Network) []string {
	return []string{
		n.ID,
		n.Name,
		n.RouterID,
		n.ZoneID,
		n.Status,
		strconv.Itoa(n.Vlan),
		n.OsNetworkID,
		strings.Join(n.Tags, ","),
	}
}

// networkShowRows renders a single network as vertical Parameter/Value rows
// covering every field the Global Router API exposes.
func networkShowRows(n v1.Network) [][]string {
	return [][]string{
		{"ID", n.ID},
		{"Name", n.Name},
		{"Router ID", n.RouterID},
		{"Zone ID", n.ZoneID},
		{"Status", n.Status},
		{"Account ID", n.AccountID},
		{"Project ID", n.ProjectID},
		{"Os Network ID", n.OsNetworkID},
		{"VLAN", strconv.Itoa(n.Vlan)},
		{"Inner VLAN", strconv.Itoa(n.InnerVlan)},
		{"Netops VLAN UUID", n.NetopsVlanUUID},
		{"Sv Network ID", n.SvNetworkID},
		{"Vdc Name", n.VdcName},
		{"Tags", strings.Join(n.Tags, ",")},
		{"Created At", n.CreatedAt},
		{"Updated At", n.UpdatedAt},
	}
}
