/*
Copyright © 2026 Kot1gruN
*/

package registry

import (
	"fmt"

	craasregistry "github.com/selectel/craas-go/pkg/v1/registry"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// registryHeader lists the compact columns shown for the table format.
var registryHeader = []string{"ID", "Name", "Status", "Size", "Size Limit", "Used"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOf(v, registryRow); rows != nil {
		return registryHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func registryRow(r *craasregistry.Registry) []string {
	return []string{
		r.ID,
		r.Name,
		string(r.Status),
		output.HumanBytes(r.Size),
		output.HumanBytes(r.SizeLimit),
		fmt.Sprintf("%.1f%%", r.Used),
	}
}
