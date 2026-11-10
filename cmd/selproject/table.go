/*
Copyright © 2026 Kot1gruN
*/
package selproject

import (
	"fmt"
	"sort"
	"strings"

	"github.com/selectel/go-selvpcclient/v5/selvpcclient/resell/v2/projects"
	"github.com/selectel/go-selvpcclient/v5/selvpcclient/resell/v2/quotas"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// projectHeader lists the compact columns shown for the table format of the
// list command. Longer formats (json/yaml) render the full records via
// projectDisplay (the SDK Project struct marshals every field to json:"-", so it
// cannot be printed directly).
var projectHeader = []string{"ID", "Name", "URL", "Enabled", "Custom URL"}

// projectShowHeader is the two-column header used for a single-project table
// result (create, update): a project has few fields and their values tend to be
// long, so a vertical parameter-per-row layout reads better.
var projectShowHeader = []string{"Parameter", "Value"}

// projectDisplay is the json/yaml projection of a project. The SDK's Project
// struct tags every field json:"-" (it has a custom UnmarshalJSON), so plain
// marshalling would yield an empty object; this type re-exposes the fields with
// explicit json tags.
type projectDisplay struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	URL       string         `json:"url"`
	Enabled   bool           `json:"enabled"`
	CustomURL string         `json:"custom_url"`
	Theme     themeDisplay   `json:"theme,omitempty"`
	Quotas    []quotaDisplay `json:"quotas,omitempty"`
}

type themeDisplay struct {
	Color string `json:"color,omitempty"`
	Logo  string `json:"logo,omitempty"`
}

type quotaDisplay struct {
	Name   string `json:"name"`
	Region string `json:"region,omitempty"`
	Zone   string `json:"zone,omitempty"`
	Value  int    `json:"value"`
	Used   int    `json:"used"`
}

func newProjectDisplay(p *projects.Project) *projectDisplay {
	if p == nil {
		return nil
	}
	d := &projectDisplay{
		ID:        p.ID,
		Name:      p.Name,
		URL:       p.URL,
		Enabled:   p.Enabled,
		CustomURL: p.CustomURL,
		Theme:     themeDisplay{Color: p.Theme.Color, Logo: p.Theme.Logo},
	}
	for _, q := range p.Quotas {
		for _, e := range q.ResourceQuotasEntities {
			d.Quotas = append(d.Quotas, quotaDisplay{
				Name:   q.Name,
				Region: e.Region,
				Zone:   e.Zone,
				Value:  e.Value,
				Used:   e.Used,
			})
		}
	}
	return d
}

func projectsToDisplay(list []*projects.Project) []*projectDisplay {
	out := make([]*projectDisplay, 0, len(list))
	for _, p := range list {
		if d := newProjectDisplay(p); d != nil {
			out = append(out, d)
		}
	}
	return out
}

func render(cmd *cobra.Command, v interface{}) error {
	// For long-form (json/yaml) output wrap the SDK records (whose fields are
	// json:"-") into the display projection; for the table format hand the raw
	// records to the table mapper.
	if !output.IsTable(cmd) {
		switch t := v.(type) {
		case []*projects.Project:
			return output.Print(cmd, projectsToDisplay(t))
		case *projects.Project:
			return output.Print(cmd, newProjectDisplay(t))
		}
	}
	return output.PrintTableValue(cmd, v, toTable)
}

// toTable renders a slice of projects as the compact list table and a single
// project as a two-column Parameter/Value table.
func toTable(v interface{}) ([]string, [][]string, error) {
	if list, ok := v.([]*projects.Project); ok {
		rows := make([][]string, 0, len(list))
		for _, p := range list {
			rows = append(rows, projectRow(p))
		}
		return projectHeader, rows, nil
	}
	if single, ok := v.(*projects.Project); ok {
		if single == nil {
			return projectShowHeader, [][]string{}, nil
		}
		return projectShowHeader, projectShowRows(*single), nil
	}
	return nil, nil, fmt.Errorf("cannot render %T as a table", v)
}

func projectRow(p *projects.Project) []string {
	if p == nil {
		return []string{"", "", "", "", ""}
	}
	return []string{
		p.ID,
		p.Name,
		p.URL,
		output.YesNo(p.Enabled),
		p.CustomURL,
	}
}

// projectShowRows renders a single project as vertical Parameter/Value rows.
func projectShowRows(p projects.Project) [][]string {
	rows := [][]string{
		{"ID", p.ID},
		{"Name", p.Name},
		{"URL", p.URL},
		{"Enabled", output.YesNo(p.Enabled)},
		{"Custom URL", p.CustomURL},
		{"Theme Color", p.Theme.Color},
		{"Theme Logo", p.Theme.Logo},
	}
	if len(p.Quotas) > 0 {
		parts := make([]string, 0, len(p.Quotas))
		for _, q := range p.Quotas {
			parts = append(parts, formatQuota(q))
		}
		sort.Strings(parts)
		rows = append(rows, []string{"Quotas", strings.Join(parts, "\n")})
	}
	return rows
}

// formatQuota renders a single quota as a compact
// "resource name: used/value (region, zone)" line.
func formatQuota(q quotas.Quota) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s:", q.Name)
	for _, e := range q.ResourceQuotasEntities {
		loc := "global"
		switch {
		case e.Region != "" && e.Zone != "":
			loc = e.Region + "/" + e.Zone
		case e.Region != "":
			loc = e.Region
		case e.Zone != "":
			loc = e.Zone
		}
		fmt.Fprintf(&b, " %s=%d/%d", loc, e.Used, e.Value)
	}
	return b.String()
}
