/*
Copyright © 2026 Kot1gruN
*/

package subnet

import (
	"fmt"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// subnetHeader lists the compact columns shown for the table format.
var subnetHeader = []string{"ID", "Name", "Network ID", "CIDR", "IP Version", "DHCP", "Gateway IP"}

// render formats v according to the --output flag.
func render(cmd *cobra.Command, v interface{}) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, v)
	}
	header, rows, err := subnetTable(v)
	if err != nil {
		return err
	}
	return output.PrintTable(cmd, header, rows)
}

// subnetTable converts a single subnet or a slice of them into table rows.
func subnetTable(v interface{}) ([]string, [][]string, error) {
	var items []subnets.Subnet
	switch data := v.(type) {
	case []subnets.Subnet:
		items = data
	case *subnets.Subnet:
		items = []subnets.Subnet{*data}
	default:
		return nil, nil, fmt.Errorf("cannot render %T as a subnet table", v)
	}

	rows := make([][]string, 0, len(items))
	for _, s := range items {
		rows = append(rows, []string{
			s.ID,
			s.Name,
			s.NetworkID,
			s.CIDR,
			strconv.Itoa(s.IPVersion),
			yesNo(s.EnableDHCP),
			s.GatewayIP,
		})
	}
	return subnetHeader, rows, nil
}

// yesNo renders a boolean as a compact yes/no value.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
