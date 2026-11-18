/*
Copyright © 2026 Kot1gruN
*/

package subnet

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// subnetHeader lists the compact columns shown for the table format.
var subnetHeader = []string{"ID", "Name", "Network ID", "CIDR", "IP Version", "DHCP", "Gateway IP"}

// subnetShowHeader is the two-column header used for the show/results table of a
// single subnet: a vertical parameter-per-row layout reads better than a wide
// single-row table.
var subnetShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow prints a single subnet with its full attribute set. Long-form
// (json/yaml) output serializes the raw subnets.Subnet value; the table format
// uses a vertical Parameter/Value layout.
func renderShow(cmd *cobra.Command, s *subnets.Subnet) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, s)
	}
	if s == nil {
		return output.PrintTable(cmd, subnetShowHeader, nil)
	}
	return output.PrintTable(cmd, subnetShowHeader, subnetShowRows(*s))
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

// subnetShowRows renders a single subnet as vertical Parameter/Value rows
// covering the fields the Neutron subnet API exposes.
func subnetShowRows(s subnets.Subnet) [][]string {
	return [][]string{
		{"ID", s.ID},
		{"Name", s.Name},
		{"Network ID", s.NetworkID},
		{"Description", emptyToDash(s.Description)},
		{"CIDR", s.CIDR},
		{"IP Version", strconv.Itoa(s.IPVersion)},
		{"Gateway IP", emptyToDash(s.GatewayIP)},
		{"DHCP", output.YesNo(s.EnableDHCP)},
		{"DNS Nameservers", joinSubnetList(s.DNSNameservers)},
		{"Allocation Pools", joinAllocationPools(s.AllocationPools)},
		{"Host Routes", joinHostRoutes(s.HostRoutes)},
		{"IPv6 Address Mode", emptyToDash(s.IPv6AddressMode)},
		{"IPv6 RA Mode", emptyToDash(s.IPv6RAMode)},
		{"Subnet Pool ID", emptyToDash(s.SubnetPoolID)},
		{"Segment ID", emptyToDash(s.SegmentID)},
		{"Tags", joinSubnetList(s.Tags)},
		{"Revision Number", strconv.Itoa(s.RevisionNumber)},
		{"Project ID", s.ProjectID},
		{"Tenant ID", s.TenantID},
		{"Created At", formatSubnetTime(s.CreatedAt)},
		{"Updated At", formatSubnetTime(s.UpdatedAt)},
	}
}

// emptyToDash returns "-" for an empty string, keeping single-line cells tidy.
func emptyToDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// joinSubnetList renders a string slice as a comma-separated list, or "-" when
// empty.
func joinSubnetList(ss []string) string {
	if len(ss) == 0 {
		return "-"
	}
	return strings.Join(ss, ", ")
}

// joinAllocationPools renders allocation pools as "start-end, ..." (or "-").
func joinAllocationPools(pools []subnets.AllocationPool) string {
	if len(pools) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(pools))
	for _, p := range pools {
		parts = append(parts, p.Start+"-"+p.End)
	}
	return strings.Join(parts, ", ")
}

// joinHostRoutes renders host routes as "destination via nexthop, ..." (or "-").
func joinHostRoutes(routes []subnets.HostRoute) string {
	if len(routes) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(routes))
	for _, r := range routes {
		parts = append(parts, r.DestinationCIDR+" via "+r.NextHop)
	}
	return strings.Join(parts, ", ")
}

// formatSubnetTime renders a timestamp, or "-" when it is the zero value.
func formatSubnetTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}
