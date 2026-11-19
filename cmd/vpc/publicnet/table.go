/*
Copyright © 2026 Kot1gruN
*/

package publicnet

import (
	"fmt"
	"strings"

	publicnetapi "github.com/selectel/public-net-api-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// portHeader lists the compact columns shown for the table format. Plain
// JSON/YAML render the full record; the table keeps only the fields most useful
// for scanning multiple public ports.
var portHeader = []string{"ID", "Project ID", "Network ID", "IP Address", "Admin Up", "Description"}

// portShowHeader is the two-column header used for the show/results table of a
// single public port: a port has many fields and their values tend to be long,
// so a vertical parameter-per-row layout reads better than a wide single-row
// table.
var portShowHeader = []string{"Parameter", "Value"}

func render(cmd *cobra.Command, v interface{}) error {
	return output.PrintTableValue(cmd, v, toTable)
}

// renderShow prints a single public port with its full attribute set. Long-form
// (json/yaml) output serializes the raw v1.Port value; the table format uses a
// vertical Parameter/Value layout so every field is visible.
func renderShow(cmd *cobra.Command, p *publicnetapi.Port) error {
	if !output.IsTable(cmd) {
		return output.Print(cmd, p)
	}
	if p == nil {
		return output.PrintTable(cmd, portShowHeader, nil)
	}
	return output.PrintTable(cmd, portShowHeader, portShowRows(*p))
}

func toTable(v interface{}) ([]string, [][]string, error) {
	if rows := output.RowsOf(v, portRow); rows != nil {
		return portHeader, rows, nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func portRow(p *publicnetapi.Port) []string {
	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	return []string{
		p.ID,
		p.ProjectID,
		p.NetworkID,
		p.IPAddress,
		output.YesNo(p.AdminStateUp),
		desc,
	}
}

// portShowRows renders a single public port as vertical Parameter/Value rows
// covering every field the Public Net API exposes.
func portShowRows(p publicnetapi.Port) [][]string {
	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	rows := [][]string{
		{"ID", p.ID},
		{"Project ID", p.ProjectID},
		{"Network ID", p.NetworkID},
		{"IP Address", p.IPAddress},
		{"Admin State Up", output.YesNo(p.AdminStateUp)},
		{"Created At", p.CreatedAt.String()},
		{"Updated At", p.UpdatedAt.String()},
	}
	if desc != "" {
		rows = append(rows, []string{"Description", desc})
	}
	if p.Subnet != "" {
		rows = append(rows, []string{"Subnet", p.Subnet})
	}
	if p.Gateway != "" {
		rows = append(rows, []string{"Gateway", p.Gateway})
	}
	if len(p.SecurityGroupIDs) > 0 {
		rows = append(rows, []string{"Security Groups", strings.Join(p.SecurityGroupIDs, "\n")})
	}
	return rows
}
