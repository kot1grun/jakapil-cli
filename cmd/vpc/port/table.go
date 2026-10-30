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

// render formats v according to the --output flag.
func render(cmd *cobra.Command, v interface{}) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, v)
	}
	header, rows, err := portTable(v)
	if err != nil {
		return err
	}
	return output.PrintTable(cmd, header, rows)
}

// portTable converts a single port or a slice of them into table rows.
func portTable(v interface{}) ([]string, [][]string, error) {
	var items []ports.Port
	switch data := v.(type) {
	case []ports.Port:
		items = data
	case *ports.Port:
		items = []ports.Port{*data}
	default:
		return nil, nil, fmt.Errorf("cannot render %T as a port table", v)
	}

	rows := make([][]string, 0, len(items))
	for _, p := range items {
		rows = append(rows, []string{
			p.ID,
			p.Name,
			p.NetworkID,
			p.MACAddress,
			p.Status,
			p.DeviceOwner,
			yesNo(p.AdminStateUp),
		})
	}
	return portHeader, rows, nil
}

// yesNo renders a boolean as a compact yes/no value.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
