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

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOfValues(v, subnetRow); rows != nil {
		return subnetHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func subnetRow(s subnets.Subnet) []string {
	return []string{
		s.ID,
		s.Name,
		s.NetworkID,
		s.CIDR,
		strconv.Itoa(s.IPVersion),
		output.YesNo(s.EnableDHCP),
		s.GatewayIP,
	}
}
