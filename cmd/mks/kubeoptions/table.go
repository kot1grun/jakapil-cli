/*
Copyright © 2026 Kot1gruN
*/

package kubeoptions

import (
	"fmt"
	"strings"

	mkskubeoptions "github.com/selectel/mks-go/pkg/v1/kubeoptions"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// kubeoptionsHeader lists the compact columns shown for the table format.
var kubeoptionsHeader = []string{"Kube Version", "Options"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOf(v, optionRow); rows != nil {
		return kubeoptionsHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func optionRow(o *mkskubeoptions.View) []string {
	return []string{o.KubeVersion, strings.Join(o.Names, ", ")}
}
