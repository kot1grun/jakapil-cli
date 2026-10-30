/*
Copyright © 2026 Kot1gruN
*/

package network

import (
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// networkHeader lists the compact columns shown for the table format. Plain
// JSON/YAML render the full record, which is verbose for lists; the table keeps
// only the fields most useful for scanning multiple networks.
var networkHeader = []string{"ID", "Name", "Status", "Shared", "Project ID"}

// render formats v according to the --output flag, falling back to the compact
// table renderer when the table format is selected.
func render(cmd *cobra.Command, v interface{}) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, v)
	}
	header, rows, err := networkTable(v)
	if err != nil {
		return err
	}
	return output.PrintTable(cmd, header, rows)
}

// networkTable converts a single network or a slice of them into table rows.
func networkTable(v interface{}) ([]string, [][]string, error) {
	var items []networks.Network
	switch data := v.(type) {
	case []networks.Network:
		items = data
	case *networks.Network:
		items = []networks.Network{*data}
	default:
		return nil, nil, fmt.Errorf("cannot render %T as a network table", v)
	}

	rows := make([][]string, 0, len(items))
	for _, n := range items {
		rows = append(rows, []string{n.ID, n.Name, n.Status, yesNo(n.Shared), n.ProjectID})
	}
	return networkHeader, rows, nil
}

// yesNo renders a boolean as a compact yes/no value.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
