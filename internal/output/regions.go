package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"jakapil-cli/internal/parallel"
)

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
			if len(rr.Rows) == 0 {
				// A region with nothing found would otherwise vanish from the
				// combined table; surface it as a notice naming the region so
				// the reader knows it was queried and returned nothing.
				fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", Yellow(cmd.ErrOrStderr(), "No resources found in region "+rr.Region))
				continue
			}
			for _, r := range rr.Rows {
				rows = append(rows, append([]string{rr.Region}, r...))
			}
		}
		return PrintTable(cmd, append([]string{"Region"}, header...), rows)
	}
	return Print(cmd, grouped)
}

// RenderRegionsBestEffort renders the aggregated "--all-regions" result produced
// by parallel.Regions for every region that succeeded. For each region that
// failed to be queried it prints a warning to stderr and excludes it from the
// output; if at least one region failed, it returns an error naming them so the
// command still exits non-zero while the data for the healthy regions is kept.
//
// The returned error is only about missing regions: a render failure is
// returned as-is and takes precedence.
func RenderRegionsBestEffort(cmd *cobra.Command, header []string, results []parallel.Result) error {
	regionRows := make([]RegionRows, 0, len(results))
	grouped := make([]RegionValue, 0, len(results))
	var failed []string
	for _, res := range results {
		if res.Err != nil {
			failed = append(failed, res.Region)
			fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", Yellow(cmd.ErrOrStderr(), "Warning: region "+res.Region+" unavailable: "+res.Err.Error()))
			continue
		}
		regionRows = append(regionRows, RegionRows{Region: res.Region, Rows: res.Rows})
		grouped = append(grouped, RegionValue{Region: res.Region, Value: res.Value})
	}

	if err := RenderRegions(cmd, header, regionRows, grouped); err != nil {
		return err
	}
	if len(failed) > 0 {
		sort.Strings(failed)
		return fmt.Errorf("could not query the following region(s): %s", strings.Join(failed, ", "))
	}
	return nil
}
