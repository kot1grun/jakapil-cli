/*
Copyright © 2026 Kot1gruN
*/

package repository

import (
	"fmt"

	craasrepository "github.com/selectel/craas-go/pkg/v1/repository"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// repositoryHeader lists the compact columns shown for the table format.
var repositoryHeader = []string{"Name", "Updated At", "Size"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOf(v, repositoryRow); rows != nil {
		return repositoryHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func repositoryRow(r *craasrepository.Repository) []string {
	return []string{r.Name, r.UpdatedAt.Format("2006-01-02 15:04:05"), output.HumanBytes(r.Size)}
}
