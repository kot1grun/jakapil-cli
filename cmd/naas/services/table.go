/*
Copyright © 2026 Kot1gruN
*/
package services

import (
	"fmt"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// serviceHeader lists the compact columns shown for the service list table
// format.
var serviceHeader = []string{"ID", "Name", "Extension", "Created At"}

// render prints the result through the --output flag: json/yaml render the raw
// v1.Service value (whose fields carry proper json tags), table delegates to the
// toTable mapper.
func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]v1.Service); ok {
		rows := make([][]string, 0, len(list))
		for _, s := range list {
			rows = append(rows, serviceRow(s))
		}
		return serviceHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func serviceRow(s v1.Service) []string {
	return []string{
		s.ID,
		s.Name,
		s.Extension,
		s.CreatedAt,
	}
}
