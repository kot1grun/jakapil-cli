package output

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRowsOf(t *testing.T) {
	type item struct{ name string }

	// Slice of pointers is flattened, nil elements skipped.
	if got := RowsOf([]*item{{"a"}, nil, {"b"}}, func(i *item) []string { return []string{i.name} }); !reflect.DeepEqual(got, [][]string{{"a"}, {"b"}}) {
		t.Fatalf("slice of pointers: got %v", got)
	}
	// Single pointer.
	if got := RowsOf(&item{"x"}, func(i *item) []string { return []string{i.name} }); !reflect.DeepEqual(got, [][]string{{"x"}}) {
		t.Fatalf("single pointer: got %v", got)
	}
	// Nil single pointer -> empty (non-nil) rows, not nil.
	if got := RowsOf((*item)(nil), func(i *item) []string { return []string{i.name} }); got == nil || !reflect.DeepEqual(got, [][]string{}) {
		t.Fatalf("nil pointer: got %v (nil=%v)", got, got == nil)
	}
	// Unrelated type -> nil sentinel so the caller can try the next mapper.
	if got := RowsOf("nope", func(i *item) []string { return nil }); got != nil {
		t.Fatalf("mismatch must be nil, got %v", got)
	}
}

func TestRowsOfValues(t *testing.T) {
	type item struct{ name string }

	if got := RowsOfValues([]item{{"a"}, {"b"}}, func(i item) []string { return []string{i.name} }); !reflect.DeepEqual(got, [][]string{{"a"}, {"b"}}) {
		t.Fatalf("slice of values: got %v", got)
	}
	if got := RowsOfValues(&item{"x"}, func(i item) []string { return []string{i.name} }); !reflect.DeepEqual(got, [][]string{{"x"}}) {
		t.Fatalf("single pointer: got %v", got)
	}
	if got := RowsOfValues("nope", func(i item) []string { return nil }); got != nil {
		t.Fatalf("mismatch must be nil, got %v", got)
	}
}

// TestPrintTableValueDispatch proves that PrintTableValue routes table output
// through the mapper and falls back to Print for long-form formats.
func TestPrintTableValueDispatch(t *testing.T) {
	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", YAML, "")
	buf := &strings.Builder{}
	cmd.SetOut(buf)

	mapper := func(v interface{}) ([]string, [][]string, error) {
		return []string{"Name"}, [][]string{{v.(string)}}, nil
	}

	// Long-form (default yaml): the mapper must not run, Print is used instead.
	if err := PrintTableValue(cmd, "value", mapper); err != nil {
		t.Fatalf("PrintTableValue(non-table): %v", err)
	}
	if buf.Len() == 0 || strings.Contains(buf.String(), "│") {
		t.Fatalf("non-table format must not render a bordered table, got:\n%s", buf.String())
	}

	// Table format: mapper runs and a bordered table is emitted.
	if err := cmd.Flags().Set("output", Table); err != nil {
		t.Fatal(err)
	}
	if err := PrintTableValue(cmd, "eta", mapper); err != nil {
		t.Fatalf("PrintTableValue(table): %v", err)
	}
	if !strings.Contains(buf.String(), "│Name") {
		t.Fatalf("expected bordered table, got:\n%s", buf.String())
	}
}
