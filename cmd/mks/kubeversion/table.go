/*
Copyright © 2026 Kot1gruN
*/

package kubeversion

import (
	"fmt"

	mkskubeversion "github.com/selectel/mks-go/pkg/v1/kubeversion"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// kubeversionHeader lists the compact columns shown for the table format.
var kubeversionHeader = []string{"Version", "Default"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOf(v, kubeversionRow); rows != nil {
		return kubeversionHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func kubeversionRow(v *mkskubeversion.View) []string {
	return []string{v.Version, output.YesNo(v.IsDefault)}
}
