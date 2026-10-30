package output

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

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
		if !strings.Contains(out, "│Region") {
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
		if strings.Contains(out, "│") {
			t.Fatalf("json output must not render a bordered table, got:\n%s", out)
		}
		if !strings.Contains(out, "\"region\": \"ru-1\"") || !strings.Contains(out, "\"region\": \"ru-2\"") {
			t.Fatalf("json output missing region grouping, got:\n%s", out)
		}
	})
}
