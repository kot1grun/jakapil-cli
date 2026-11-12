/*
Copyright © 2026 Kot1gruN
*/

package floatingip

import (
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// floatingIPHeader lists the compact columns shown for the table format. Plain
// JSON/YAML render the full record; the table keeps only the fields most useful
// for scanning multiple floating IPs.
var floatingIPHeader = []string{"ID", "Floating IP", "Fixed IP", "Status", "Port ID", "External Network ID"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOfValues(v, floatingIPRow); rows != nil {
		return floatingIPHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func floatingIPRow(f floatingips.FloatingIP) []string {
	return []string{f.ID, f.FloatingIP, f.FixedIP, f.Status, f.PortID, f.FloatingNetworkID}
}
