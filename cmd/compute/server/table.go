/*
Copyright © 2026 Kot1gruN
*/
package server

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// serverHeader lists the compact columns shown for the table format of the
// list command. Plain JSON/YAML render the full record, which is verbose for
// lists; the table keeps only the fields most useful for scanning multiple
// servers.
var serverHeader = []string{"ID", "Name", "Status", "Flavor ID", "Image ID", "IPs", "Key Name", "AZ", "Created"}

// serverShowHeader is the two-column header used for a single-server table
// result (show, create, update). A server is rendered vertically, one parameter
// per row, so long properties don't overload a single wide row.
var serverShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow renders a single server result (show, create, update). The table
// format uses the vertical Parameter/Value layout; the long formats carry the
// full record.
func renderShow(cmd *cobra.Command, s *servers.Server) error {
	if output.IsTable(cmd) {
		return output.PrintTable(cmd, serverShowHeader, serverShowRows(s))
	}
	return output.Print(cmd, s)
}

// toTable renders a slice of servers as the compact list table.
func toTable(v interface{}) ([]string, [][]string, error) {
	list, ok := v.([]servers.Server)
	if !ok {
		return nil, nil, fmt.Errorf("cannot render %T as a table", v)
	}
	rows := make([][]string, 0, len(list))
	for _, s := range list {
		rows = append(rows, serverRow(s))
	}
	return serverHeader, rows, nil
}

func serverRow(s servers.Server) []string {
	return []string{
		s.ID,
		s.Name,
		s.Status,
		resourceID(s.Flavor),
		resourceID(s.Image),
		addressString(s.Addresses),
		s.KeyName,
		s.AvailabilityZone,
		formatTime(s.Created),
	}
}

// serverShowRows renders a single server as vertical Parameter/Value rows.
func serverShowRows(s *servers.Server) [][]string {
	if s == nil {
		return [][]string{}
	}
	rows := [][]string{
		{"ID", s.ID},
		{"Name", s.Name},
		{"Status", s.Status},
		{"Flavor ID", resourceID(s.Flavor)},
		{"Image ID", resourceID(s.Image)},
		{"Key Name", s.KeyName},
		{"Availability Zone", s.AvailabilityZone},
		{"Host", s.Host},
		{"Instance Name", s.InstanceName},
		{"Hypervisor", s.HypervisorHostname},
		{"VM State", s.VmState},
		{"Task State", s.TaskState},
		{"Power State", s.PowerState.String()},
		{"Created", formatTime(s.Created)},
		{"Updated", formatTime(s.Updated)},
		{"IPs", addressString(s.Addresses)},
		{"Access IPv4", s.AccessIPv4},
		{"Access IPv6", s.AccessIPv6},
		{"Security Groups", securityGroupString(s.SecurityGroups)},
		{"Attached Volumes", attachedVolumeString(s.AttachedVolumes)},
		{"Tags", tagsString(s.Tags)},
	}
	if len(s.Metadata) > 0 {
		parts := make([]string, 0, len(s.Metadata))
		for _, k := range sortedKeys(s.Metadata) {
			parts = append(parts, fmt.Sprintf("%s=%s", k, s.Metadata[k]))
		}
		rows = append(rows, []string{"Metadata", strings.Join(parts, "\n")})
	}
	return rows
}

// resourceID extracts the "id" string from a gophercloud server resource map
// (the flavor/image inline object), returning an empty string when absent.
func resourceID(m map[string]any) string {
	if m == nil {
		return ""
	}
	if id, ok := m["id"].(string); ok {
		return id
	}
	return ""
}

// addressString flattens the server addresses map (keyed by network name) into
// a comma-separated list of IPs.
func addressString(addrs map[string]any) string {
	var ips []string
	for _, network := range addrs {
		switch typed := network.(type) {
		case []any:
			for _, entry := range typed {
				if m, ok := entry.(map[string]any); ok {
					if a, ok := m["addr"].(string); ok {
						ips = append(ips, a)
					}
				}
			}
		case []map[string]any:
			for _, m := range typed {
				if a, ok := m["addr"].(string); ok {
					ips = append(ips, a)
				}
			}
		}
	}
	sort.Strings(ips)
	return strings.Join(ips, ", ")
}

// securityGroupString renders the server security groups as a comma-separated
// list of names.
func securityGroupString(sgs []map[string]any) string {
	if len(sgs) == 0 {
		return ""
	}
	names := make([]string, 0, len(sgs))
	for _, sg := range sgs {
		if n, ok := sg["name"].(string); ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// attachedVolumeString renders the attached volume IDs as a comma-separated list.
func attachedVolumeString(vols []servers.AttachedVolume) string {
	if len(vols) == 0 {
		return ""
	}
	ids := make([]string, 0, len(vols))
	for _, v := range vols {
		if v.ID != "" {
			ids = append(ids, v.ID)
		}
	}
	return strings.Join(ids, ", ")
}

// tagsString renders the optional server tags as a comma-separated list.
func tagsString(tags *[]string) string {
	if tags == nil {
		return ""
	}
	return strings.Join(*tags, ", ")
}

// formatTime formats a time.Time in a compact, column-friendly layout. The
// zero time is rendered as an empty string.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// sortedKeys returns the keys of m in ascending order.
func sortedKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
