/*
Copyright © 2026 Kot1gruN
*/

package port

import (
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// portHeader lists the compact columns shown for the table format.
var portHeader = []string{"ID", "Name", "Network ID", "MAC", "Status", "Device Owner", "Admin Up"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOfValues(v, portRow); rows != nil {
		return portHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func portRow(p ports.Port) []string {
	return []string{
		p.ID,
		p.Name,
		p.NetworkID,
		p.MACAddress,
		p.Status,
		p.DeviceOwner,
		output.YesNo(p.AdminStateUp),
	}
}
