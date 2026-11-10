/*
Copyright © 2026 Kot1gruN
*/
package volume

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// volumeTypeHeader lists the compact columns shown for the table format of the
// volume type list command. Plain JSON/YAML render the full record.
var volumeTypeHeader = []string{"ID", "Name", "Description", "Is Public", "QoS Spec ID"}

// volumeTypeShowHeader is the two-column header used for the single volume type
// table result (show).
var volumeTypeShowHeader = []string{"Parameter", "Value"}

func renderTypes(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTypeTable)
}

// renderTypeShow renders a single volume type result (show). The table format
// uses the vertical Parameter/Value layout; the long formats carry the full
// record.
func renderTypeShow(cmd *cobra.Command, vt *volumetypes.VolumeType) error {
	if output.IsTable(cmd) {
		return output.PrintTable(cmd, volumeTypeShowHeader, volumeTypeShowRows(vt))
	}
	return output.Print(cmd, vt)
}

// toTypeTable renders a slice of volume types as the compact list table.
func toTypeTable(v interface{}) ([]string, [][]string, error) {
	list, ok := v.([]volumetypes.VolumeType)
	if !ok {
		return nil, nil, fmt.Errorf("cannot render %T as a table", v)
	}
	rows := make([][]string, 0, len(list))
	for _, vt := range list {
		rows = append(rows, volumeTypeRow(vt))
	}
	return volumeTypeHeader, rows, nil
}

func volumeTypeRow(vt volumetypes.VolumeType) []string {
	return []string{
		vt.ID,
		vt.Name,
		vt.Description,
		output.YesNo(vt.IsPublic),
		vt.QosSpecID,
	}
}

// volumeTypeShowRows renders a single volume type as vertical
// Parameter/Value rows.
func volumeTypeShowRows(vt *volumetypes.VolumeType) [][]string {
	if vt == nil {
		return [][]string{}
	}
	rows := [][]string{
		{"ID", vt.ID},
		{"Name", vt.Name},
		{"Description", vt.Description},
		{"Is Public", output.YesNo(vt.IsPublic)},
		{"QoS Spec ID", vt.QosSpecID},
	}
	if len(vt.ExtraSpecs) > 0 {
		parts := make([]string, 0, len(vt.ExtraSpecs))
		for _, k := range sortedKeys(vt.ExtraSpecs) {
			parts = append(parts, fmt.Sprintf("%s=%s", k, vt.ExtraSpecs[k]))
		}
		rows = append(rows, []string{"Extra Specs", strings.Join(parts, "\n")})
	}
	return rows
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
