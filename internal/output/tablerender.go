package output

import "github.com/spf13/cobra"

// RowMapper converts a command result (a single resource or a slice of them)
// into a table header and data rows. It is the per-command companion of
// PrintTableValue.
type RowMapper func(v interface{}) ([]string, [][]string, error)

// PrintTableValue prints v according to the --output flag. Long-form formats
// (json/yaml) fall through to Print; the "table" format is delegated to mapper,
// which returns the header and rows for PrintTable.
func PrintTableValue(cmd *cobra.Command, v interface{}, mapper RowMapper) error {
	if !IsTable(cmd) {
		return Print(cmd, v)
	}
	header, rows, err := mapper(v)
	if err != nil {
		return err
	}
	return PrintTable(cmd, header, rows)
}

// RowsOf normalizes a pointer-based result — a slice of *T or a single *T — into
// table rows with row. It returns nil for any other type, so callers can try
// several row builders in sequence (e.g. for several resource kinds in one
// package). Nil elements and a nil single pointer yield no rows, never an error.
func RowsOf[T any](v interface{}, row func(*T) []string) [][]string {
	switch d := v.(type) {
	case []*T:
		rows := make([][]string, 0, len(d))
		for _, item := range d {
			if item != nil {
				rows = append(rows, row(item))
			}
		}
		return rows
	case *T:
		if d == nil {
			return [][]string{}
		}
		return [][]string{row(d)}
	}
	return nil
}

// RowsOfValues is the value-based counterpart of RowsOf: it handles a slice of
// T values or a single *T. Use it for SDKs that return value slices (e.g.
// gophercloud).
func RowsOfValues[T any](v interface{}, row func(T) []string) [][]string {
	switch d := v.(type) {
	case []T:
		rows := make([][]string, 0, len(d))
		for _, item := range d {
			rows = append(rows, row(item))
		}
		return rows
	case *T:
		if d == nil {
			return [][]string{}
		}
		return [][]string{row(*d)}
	}
	return nil
}
