package output

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"jakapil-cli/internal/parallel"
)

// errUnavailable is a sentinel used to simulate a region whose endpoint cannot
// be queried in RenderRegionsBestEffort tests.
var errUnavailable = errors.New("endpoint unavailable")

// TestRenderRegions proves that --all-regions output prepends a Region column
// for the table format and switches to grouped long-form output otherwise.
func TestRenderRegions(t *testing.T) {
	header := []string{"ID"}
	regionRows := []RegionRows{
		{Region: "ru-1", Rows: [][]string{{"a"}}},
		{Region: "ru-2", Rows: [][]string{{"b"}, {"c"}}},
	}
	grouped := []RegionValue{
		{Region: "ru-1", Value: "a"},
		{Region: "ru-2", Value: "b"},
	}

	newCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "leaf"}
		cmd.Flags().StringP("output", "o", YAML, "")
		cmd.SetOut(&strings.Builder{})
		return cmd
	}

	t.Run("table prepends a Region column to every row", func(t *testing.T) {
		cmd := newCmd()
		if err := cmd.Flags().Set("output", Table); err != nil {
			t.Fatal(err)
		}
		if err := RenderRegions(cmd, header, regionRows, grouped); err != nil {
			t.Fatalf("RenderRegions(table): %v", err)
		}
		out := cmd.OutOrStdout().(*strings.Builder).String()
		if !strings.Contains(out, "║Region") {
			t.Fatalf("expected a Region column, got:\n%s", out)
		}
		// ru-1 feeds 1 row, ru-2 feeds 2 rows — each must carry its region cell.
		if got := strings.Count(out, "ru-1"); got != 1 {
			t.Fatalf("ru-1 appears %d times, want 1 (header + single row group marker); got:\n%s", got, out)
		}
		if got := strings.Count(out, "ru-2"); got != 2 {
			t.Fatalf("ru-2 appears %d times, want 2; got:\n%s", got, out)
		}
	})

	t.Run("long-form json renders grouped RegionValues", func(t *testing.T) {
		cmd := newCmd()
		if err := cmd.Flags().Set("output", "json"); err != nil {
			t.Fatal(err)
		}
		if err := RenderRegions(cmd, header, regionRows, grouped); err != nil {
			t.Fatalf("RenderRegions(json): %v", err)
		}
		out := cmd.OutOrStdout().(*strings.Builder).String()
		if strings.Contains(out, "║") {
			t.Fatalf("json output must not render a bordered table, got:\n%s", out)
		}
		if !strings.Contains(out, "\"region\": \"ru-1\"") || !strings.Contains(out, "\"region\": \"ru-2\"") {
			t.Fatalf("json output missing region grouping, got:\n%s", out)
		}
	})
}

// TestRenderRegionsEmptyRegion verifies that a region with no rows yields a
// per-region warning in the table format instead of silently vanishing, and
// does not leak into the combined table.
func TestRenderRegionsEmptyRegion(t *testing.T) {
	header := []string{"ID"}
	regionRows := []RegionRows{
		{Region: "ru-1", Rows: [][]string{{"a"}}},
		{Region: "ru-2", Rows: [][]string{}},
	}
	grouped := []RegionValue{
		{Region: "ru-1", Value: "a"},
		{Region: "ru-2", Value: nil},
	}

	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", Table, "")
	var out, warn strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&warn)

	if err := RenderRegions(cmd, header, regionRows, grouped); err != nil {
		t.Fatalf("RenderRegions: %v", err)
	}
	if !strings.Contains(warn.String(), "ru-2") || !strings.Contains(warn.String(), "no resources found") {
		t.Fatalf("expected a per-region empty warning on stderr, got: %q", warn.String())
	}
	// The empty region must not become a table row; only ru-1 renders.
	if got := strings.Count(out.String(), "ru-1"); got != 1 {
		t.Fatalf("ru-1 appears %d times, want 1; got:\n%s", got, out.String())
	}
	if strings.Contains(out.String(), "ru-2") {
		t.Fatalf("empty region ru-2 leaked into the table output:\n%s", out.String())
	}

	// Long-form output keeps the empty region in the grouped result unchanged.
	cmd2 := &cobra.Command{Use: "leaf"}
	cmd2.Flags().StringP("output", "o", YAML, "")
	var out2 strings.Builder
	cmd2.SetOut(&out2)
	if err := RenderRegions(cmd2, header, regionRows, grouped); err != nil {
		t.Fatalf("RenderRegions(yaml): %v", err)
	}
	if !strings.Contains(out2.String(), "ru-2") {
		t.Fatalf("long-form output must keep the empty region, got:\n%s", out2.String())
	}
}

// TestRenderRegionsBestEffort verifies that a failed region is skipped with a
// warning on stderr, the healthy regions are still rendered, and the command
// reports an error naming the failed region so it exits non-zero.
func TestRenderRegionsBestEffort(t *testing.T) {
	header := []string{"ID"}
	results := []parallel.Result{
		{Region: "ru-1", Rows: [][]string{{"a"}}, Value: "a"},
		{Region: "ru-8", Err: errUnavailable},
		{Region: "ru-3", Rows: [][]string{{"c"}}, Value: "c"},
	}

	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", Table, "")
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})

	err := RenderRegionsBestEffort(cmd, header, results)
	if err == nil {
		t.Fatal("expected an error naming the unavailable region, got nil")
	}
	if !strings.Contains(err.Error(), "ru-8") {
		t.Fatalf("error %q should name ru-8", err)
	}

	// The healthy regions ru-1 and ru-3 must still appear in the table, while
	// the failed ru-8 is excluded from the data.
	out := cmd.OutOrStdout().(*strings.Builder).String()
	if !strings.Contains(out, "ru-1") || !strings.Contains(out, "ru-3") {
		t.Fatalf("expected healthy regions in output, got:\n%s", out)
	}
	// ru-1 and ru-3 each carry exactly one data row besides their header cell.
	if got := strings.Count(out, "ru-1"); got != 1 {
		t.Fatalf("ru-1 appears %d times, want 1; got:\n%s", got, out)
	}
	if got := strings.Count(out, "ru-3"); got != 1 {
		t.Fatalf("ru-3 appears %d times, want 1; got:\n%s", got, out)
	}

	// ru-8 should only surface in the stderr warning, not in the table.
	warn := cmd.ErrOrStderr().(*strings.Builder).String()
	if !strings.Contains(warn, "ru-8") {
		t.Fatalf("expected a warning about ru-8 on stderr, got:\n%q", warn)
	}
	if strings.Contains(out, "ru-8") {
		t.Fatalf("failed region ru-8 leaked into the table output:\n%s", out)
	}
}
