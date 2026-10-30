/*
Copyright © 2026 Kot1gruN
*/

package node

import (
	"fmt"

	mksnode "github.com/selectel/mks-go/pkg/v1/node"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// nodeHeader lists the compact columns shown for the table format.
var nodeHeader = []string{"ID", "Hostname", "IP", "Nodegroup ID", "OS Server ID"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOf(v, nodeRow); rows != nil {
		return nodeHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func nodeRow(n *mksnode.View) []string {
	return []string{n.ID, n.Hostname, n.IP, n.NodegroupID, n.OSServerID}
}
