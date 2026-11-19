/*
Copyright © 2026 Kot1gruN
*/
package publicsubnet

import (
	"fmt"
	"strconv"

	"github.com/selectel/go-selvpcclient/v5/selvpcclient/resell/v2/subnets"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// subnetHeader lists the compact columns shown for the table format. Plain
// JSON/YAML render the full record.
var subnetHeader = []string{"ID", "Status", "Region", "CIDR", "Network ID", "Subnet ID", "Project ID"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOf(v, subnetRow); rows != nil {
		return subnetHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func subnetRow(s *subnets.Subnet) []string {
	if s == nil {
		return []string{"", "", "", "", "", "", ""}
	}
	return []string{
		strconv.Itoa(s.ID),
		s.Status,
		s.Region,
		emptyToDash(s.CIDR),
		dashOrUnassigned(s.NetworkID),
		dashOrUnassigned(s.SubnetID),
		dashOrUnassigned(s.ProjectID),
	}
}

// emptyToDash returns "-" for an empty string, keeping single-line cells tidy.
func emptyToDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// dashOrUnassigned returns "unassigned" for an identifier that has not been
// created yet (a freshly requested subnet), keeping the distinction from a
// plain empty value explicit.
func dashOrUnassigned(s string) string {
	if s == "" {
		return "unassigned"
	}
	return s
}
