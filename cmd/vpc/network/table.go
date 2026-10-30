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

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOfValues(v, networkRow); rows != nil {
		return networkHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func networkRow(n networks.Network) []string {
	return []string{n.ID, n.Name, n.Status, output.YesNo(n.Shared), n.ProjectID}
}
