/*
Copyright © 2026 Kot1gruN
*/

package flavor

import (
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// flavorHeader lists the compact columns shown for the table format. Plain
// JSON/YAML render the full record, which is verbose for lists; the table keeps
// only the fields most useful for scanning multiple flavors.
var flavorHeader = []string{"ID", "Name", "RAM (MB)", "VCPUs", "Disk (GB)", "Public", "Description"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOfValues(v, flavorRow); rows != nil {
		return flavorHeader, rows, nil
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
