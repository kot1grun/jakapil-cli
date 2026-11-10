/*
Copyright © 2026 Kot1gruN
*/
package volume

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// volumeHeader lists the compact columns shown for the table format of the
// list command. Plain JSON/YAML render the full record; the table keeps only
// the fields most useful for scanning multiple volumes.
var volumeHeader = []string{"ID", "Name", "Status", "Size (GB)", "Volume Type", "AZ", "Attached To", "Bootable", "Encrypted"}

// volumeShowHeader is the two-column header used for a single-volume table
// result (show, create, update). A volume is rendered vertically, one
// parameter per row, so long properties don't overload a single wide row.
var volumeShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow renders a single volume result (show, create, update). The table
// format uses the vertical Parameter/Value layout; the long formats carry the
// full record.
func renderShow(cmd *cobra.Command, vol *volumes.Volume) error {
	if output.IsTable(cmd) {
		return output.PrintTable(cmd, volumeShowHeader, volumeShowRows(vol))
	}
	return output.Print(cmd, vol)
}

// toTable renders a slice of volumes as the compact list table.
func toTable(v interface{}) ([]string, [][]string, error) {
	list, ok := v.([]volumes.Volume)
	if !ok {
		return nil, nil, fmt.Errorf("cannot render %T as a table", v)
	}
	rows := make([][]string, 0, len(list))
	for _, vol := range list {
		rows = append(rows, volumeRow(vol))
	}
	return volumeHeader, rows, nil
}

func volumeRow(vol volumes.Volume) []string {
	return []string{
		vol.ID,
		vol.Name,
		vol.Status,
		fmt.Sprintf("%d", vol.Size),
		vol.VolumeType,
		vol.AvailabilityZone,
		attachedServers(vol),
		output.YesNo(strings.EqualFold(vol.Bootable, "true")),
		output.YesNo(vol.Encrypted),
	}
}

// attachedServers returns a comma-separated list of server IDs the volume is
// attached to, or "-" when it has no attachments.
func attachedServers(vol volumes.Volume) string {
	if len(vol.Attachments) == 0 {
		return "-"
	}
	ids := make([]string, 0, len(vol.Attachments))
	for _, a := range vol.Attachments {
		if a.ServerID != "" {
			ids = append(ids, a.ServerID)
		}
	}
	return strings.Join(ids, ", ")
}

// volumeShowRows renders a single volume as vertical Parameter/Value rows.
func volumeShowRows(vol *volumes.Volume) [][]string {
	if vol == nil {
		return [][]string{}
	}
	rows := [][]string{
		{"ID", vol.ID},
		{"Name", vol.Name},
		{"Status", vol.Status},
		{"Size (GB)", fmt.Sprintf("%d", vol.Size)},
		{"Volume Type", vol.VolumeType},
		{"Availability Zone", vol.AvailabilityZone},
		{"Bootable", output.YesNo(strings.EqualFold(vol.Bootable, "true"))},
		{"Encrypted", output.YesNo(vol.Encrypted)},
		{"Description", vol.Description},
		{"Attached To", attachedServers(*vol)},
		{"Snapshot ID", vol.SnapshotID},
		{"Source Vol ID", vol.SourceVolID},
		{"Backup ID", stringOrEmpty(vol.BackupID)},
		{"Tenant ID", vol.TenantID},
		{"User ID", vol.UserID},
		{"Created At", vol.CreatedAt.Format("2006-01-02 15:04:05")},
		{"Updated At", vol.UpdatedAt.Format("2006-01-02 15:04:05")},
	}
	if len(vol.Metadata) > 0 {
		parts := make([]string, 0, len(vol.Metadata))
		for k, v := range vol.Metadata {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
		sort.Strings(parts)
		rows = append(rows, []string{"Metadata", strings.Join(parts, "\n")})
	}
	return rows
}

// stringOrEmpty dereferences a string pointer, returning an empty string for a
// nil pointer (used for the volume BackupID field which is *string).
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
