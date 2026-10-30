package output

import "github.com/spf13/cobra"

// RegionValue pairs a region with the value returned by listing that region. It
// is what "--all-regions" renders for the long-form (json/yaml) formats, so the
// grouped output stays orderable and the region is explicit.
type RegionValue struct {
	Region string      `json:"region"`
	Value  interface{} `json:"items"`
}

// RegionRows holds pre-rendered table rows for one region, without the region
// column (RenderRegions prepends it to every row).
type RegionRows struct {
	Region string
	Rows   [][]string
}

// RenderRegions renders an aggregated "--all-regions" result according to the
// --output flag:
//
//   - table: prepends a "Region" column and prints the combined grid;
//   - json/yaml: prints the grouped output ([]RegionValue) via Print.
//
// Regions and their rows are passed in the order produced by the caller (the
// keystone region discovery sort them by name), so the output is deterministic.
func RenderRegions(cmd *cobra.Command, header []string, regionRows []RegionRows, grouped []RegionValue) error {
	if IsTable(cmd) {
		rows := make([][]string, 0)
		for _, rr := range regionRows {
			for _, r := range rr.Rows {
				rows = append(rows, append([]string{rr.Region}, r...))
			}
		}
		return PrintTable(cmd, append([]string{"Region"}, header...), rows)
	}
	return Print(cmd, grouped)
}
