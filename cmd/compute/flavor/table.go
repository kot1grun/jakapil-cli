/*
Copyright © 2026 Kot1gruN
*/

package flavor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// flavorHeader lists the compact columns shown for the table format of the
// list command. Plain JSON/YAML render the full record, which is verbose for
// lists; the table keeps only the fields most useful for scanning multiple
// flavors.
var flavorHeader = []string{"ID", "Name", "RAM (MB)", "VCPUs", "Disk (GB)", "Public", "Description"}

// flavorShowHeader is the two-column header used for a single-flavor table
// result (show, create, update). A flavor is rendered vertically, one
// parameter per row, so long properties don't overload a single wide row.
var flavorShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// toTable renders a slice of flavors as the compact list table and a single
// flavor (backend returns *flavors.Flavor from Create/Update/Get) as a
// two-column Parameter/Value table.
func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]flavors.Flavor); ok {
		rows := make([][]string, 0, len(list))
		for _, f := range list {
			rows = append(rows, flavorRow(f))
		}
		return flavorHeader, rows, nil
	}
	if single, ok := v.(*flavors.Flavor); ok {
		if single == nil {
			return flavorShowHeader, [][]string{}, nil
		}
		return flavorShowHeader, flavorShowRows(*single), nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func flavorRow(f flavors.Flavor) []string {
	return []string{
		f.ID,
		f.Name,
		fmt.Sprintf("%d", f.RAM),
		fmt.Sprintf("%d", f.VCPUs),
		fmt.Sprintf("%d", f.Disk),
		output.YesNo(f.IsPublic),
		f.Description,
	}
}

// flavorShowRows renders a single flavor as vertical Parameter/Value rows. The
// extra-specs (properties) are grouped into a single "Properties" row whose
// value lists each key=value pair one per line (sorted by key).
func flavorShowRows(f flavors.Flavor) [][]string {
	rows := [][]string{
		{"ID", f.ID},
		{"Name", f.Name},
		{"RAM (MB)", fmt.Sprintf("%d", f.RAM)},
		{"VCPUs", fmt.Sprintf("%d", f.VCPUs)},
		{"Disk (GB)", fmt.Sprintf("%d", f.Disk)},
		{"Public", output.YesNo(f.IsPublic)},
		{"Description", f.Description},
	}
	if len(f.ExtraSpecs) > 0 {
		parts := make([]string, 0, len(f.ExtraSpecs))
		for _, k := range sortedKeys(f.ExtraSpecs) {
			parts = append(parts, fmt.Sprintf("%s=%s", k, f.ExtraSpecs[k]))
		}
		rows = append(rows, []string{"Properties", strings.Join(parts, "\n")})
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
