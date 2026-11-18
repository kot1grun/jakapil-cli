/*
Copyright © 2026 Kot1gruN
*/

package network

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// networkHeader lists the compact columns shown for the table format. Plain
// JSON/YAML render the full record, which is verbose for lists; the table keeps
// only the fields most useful for scanning multiple networks.
var networkHeader = []string{"ID", "Name", "Status", "Shared", "Project ID"}

// networkShowHeader is the two-column header used for the show/results table of a
// single network: a vertical parameter-per-row layout reads better than a wide
// single-row table.
var networkShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow prints a single network with its full attribute set. Long-form
// (json/yaml) output serializes the raw networks.Network value; the table format
// uses a vertical Parameter/Value layout.
func renderShow(cmd *cobra.Command, n *networks.Network) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, n)
	}
	if n == nil {
		return output.PrintTable(cmd, networkShowHeader, nil)
	}
	return output.PrintTable(cmd, networkShowHeader, networkShowRows(*n))
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

// networkShowRows renders a single network as vertical Parameter/Value rows
// covering the fields the Neutron network API exposes.
func networkShowRows(n networks.Network) [][]string {
	return [][]string{
		{"ID", n.ID},
		{"Name", n.Name},
		{"Status", n.Status},
		{"Admin State", output.YesNo(n.AdminStateUp)},
		{"Shared", output.YesNo(n.Shared)},
		{"Description", emptyToDash(n.Description)},
		{"Project ID", n.ProjectID},
		{"Tenant ID", n.TenantID},
		{"Tags", joinList(n.Tags)},
		{"Availability Zone Hints", joinList(n.AvailabilityZoneHints)},
		{"Subnets", joinList(n.Subnets)},
		{"Revision Number", strconv.Itoa(n.RevisionNumber)},
		{"Created At", formatNetTime(n.CreatedAt)},
		{"Updated At", formatNetTime(n.UpdatedAt)},
	}
}

// emptyToDash returns "-" for an empty string, keeping single-line cells tidy.
func emptyToDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// joinList renders a string slice as a comma-separated list, or "-" when empty.
func joinList(ss []string) string {
	if len(ss) == 0 {
		return "-"
	}
	return strings.Join(ss, ", ")
}

// formatNetTime renders a timestamp, or "-" when it is the zero value.
func formatNetTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}
