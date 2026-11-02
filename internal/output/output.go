/*
Copyright © 2026 Kot1gruN
*/
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
	"golang.org/x/term"
)

const (
	// JSON represents the JSON output format.
	JSON = "json"

	// YAML represents the YAML output format.
	YAML = "yaml"

	// Table represents the compact tabular output format (rivo/tview).
	Table = "table"
)

// Print renders v to the command output using the value of the persistent
// --output flag on cmd. Supported formats are "json" and "yaml"; an empty
// format falls back to printing v (or each element of a slice/array) with %+v.
func Print(cmd *cobra.Command, v interface{}) error {
	format, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()

	switch format {
	case "":
		return printPlain(w, v)
	case JSON:
		return renderJSON(w, v)
	case YAML:
		return renderYAML(w, v)
	default:
		return fmt.Errorf("unsupported output format %q: use %q or %q", format, JSON, YAML)
	}
}

// printPlain prints a single value with %+v or, for slices/arrays, each element.
func printPlain(w io.Writer, v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		for i := 0; i < rv.Len(); i++ {
			_, err := fmt.Fprintf(w, "%+v\n", rv.Index(i).Interface())
			if err != nil {
				return err
			}
		}
		return nil
	}
	_, err := fmt.Fprintf(w, "%+v\n", v)
	return err
}

func renderJSON(w io.Writer, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func renderYAML(w io.Writer, v interface{}) error {
	// Marshal through JSON first so the YAML mirrors the JSON representation,
	// honouring json tags (including fields tagged json:"-" and omitempty) and
	// avoiding the rendering of anonymous embedded structs (e.g. mks-go's
	// BaseView) as a nested key.
	jb, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var data interface{}
	if err := yaml.Unmarshal(jb, &data); err != nil {
		return err
	}
	b, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// IsTable reports whether the --output flag on cmd is set to the compact
// tabular format. Long-form formats (JSON/YAML) carry the full record set,
// which gets verbose for list/table data, so callers route to PrintTable.
func IsTable(cmd *cobra.Command) bool {
	format, _ := cmd.Flags().GetString("output")
	return format == Table
}

// YesNo renders a boolean as a compact "yes"/"no" value for table cells.
func YesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// ANSI escapes used to highlight non-fatal notices (e.g. an empty result set)
// when stdout is a terminal. They are omitted when output is redirected or
// piped, so the stream stays plain and parseable (same policy as cmd/errorout).
const (
	ansiYellow = "\x1b[1;33m" // bold yellow
	ansiReset  = "\x1b[0m"
)

// Yellow wraps s in bold ANSI yellow when w is attached to a terminal and
// returns s unchanged otherwise. w must be the writer the coloured text is
// actually going to, so buffer-backed (non-file) writers get plain text.
func Yellow(w io.Writer, s string) string {
	if f, ok := w.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		return ansiYellow + s + ansiReset
	}
	return s
}

// HumanBytes renders a byte count in a human-readable decimal form (KiB/MiB/
// GiB/...), used to keep table cells short.
func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// PrintTable renders header and rows as an aligned table to the command output.
// The table is built on top of rivo/tview's Table widget and rendered through a
// tcell simulation screen, so column widths and alignment match what tview
// itself would draw in a full TUI.
func PrintTable(cmd *cobra.Command, header []string, rows [][]string) error {
	if len(rows) == 0 {
		// A header-only table carries no information for a human reader, so a
		// plain notice is more useful. Long-form formats (json/yaml) keep the
		// empty collection untouched and never reach this branch.
		w := cmd.OutOrStdout()
		_, err := fmt.Fprintln(w, Yellow(w, "No resources found."))
		return err
	}
	text, err := RenderTable(header, rows)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), text)
	return err
}

// RenderTable builds an aligned plain-text table out of a header row and data
// rows using rivo/tview. The widget is drawn onto an off-screen tcell
// simulation screen and the resulting cells are reassembled into text lines.
func RenderTable(header []string, rows [][]string) (string, error) {
	columns := len(header)
	if columns == 0 {
		return "", nil
	}

	table := tview.NewTable()
	table.SetBorder(false)
	table.SetBorders(true)
	for c, h := range header {
		table.SetCell(0, c, tview.NewTableCell(h).SetTextColor(tcell.ColorWhite).SetAttributes(tcell.AttrBold))
	}
	for r, row := range rows {
		for c := 0; c < columns; c++ {
			text := ""
			if c < len(row) {
				text = row[c]
			}
			table.SetCell(r+1, c, tview.NewTableCell(text))
		}
	}

	// Compute the exact space the bordered table needs so Draw won't clip
	// anything. Each logical row takes two screen rows (a border line and a
	// text line) plus one bottom border; each column is surrounded by a border
	// on either side plus its own right edge line.
	colWidth := make([]int, columns)
	for c, h := range header {
		colWidth[c] = runewidth.StringWidth(h)
	}
	for _, row := range rows {
		for c := 0; c < columns; c++ {
			text := ""
			if c < len(row) {
				text = row[c]
			}
			if w := runewidth.StringWidth(text); w > colWidth[c] {
				colWidth[c] = w
			}
		}
	}
	sum := 0
	for _, w := range colWidth {
		sum += w
	}
	tableRows := len(rows) + 1 // header + data rows
	width := sum + columns + 2 // 2 edge borders + one border line per column separator
	height := 2*tableRows + 1  // each logical row (border+text) + bottom border

	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		return "", err
	}
	defer screen.Fini()
	screen.SetSize(width, height)
	table.SetRect(0, 0, width, height)
	table.Draw(screen)
	screen.Sync()

	cells, _, _ := screen.GetContents()
	return doubleOuterBorders(extractTable(cells, width, height)), nil
}

// doubleOuterBorders promotes the outer frame of a rendered table to heavy
// box-drawing characters while keeping the internal grid separators thin. tview
// draws the whole grid with single lines (┌─┬┐ │ ├┼┤ └┴┘); this post-processor
// rewrites every line according to its first cell so the perimeter becomes
// double (╔═╤╗ ║ ╠╣ ╚═╧╝) and only the internal column separators, row rules and
// their crossings remain single (│ ─ ┼).
func doubleOuterBorders(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, line)
			continue
		}
		r := []rune(line)
		switch r[0] {
		case '┌': // outer top border: double horizontal, junction with thin column becomes ╤
			for i, ch := range r {
				switch ch {
				case '─':
					r[i] = '═'
				case '┬':
					r[i] = '╤'
				case '┌':
					r[i] = '╔'
				case '┐':
					r[i] = '╗'
				}
			}
		case '├': // inner horizontal rule: only the side ends meet the double frame
			for i, ch := range r {
				switch ch {
				case '├':
					r[i] = '╠'
				case '┤':
					r[i] = '╣'
				}
			}
		case '└': // outer bottom border: double horizontal, junction with thin column becomes ╧
			for i, ch := range r {
				switch ch {
				case '─':
					r[i] = '═'
				case '┴':
					r[i] = '╧'
				case '└':
					r[i] = '╚'
				case '┘':
					r[i] = '╝'
				}
			}
		case '│': // content row: double the two edge cells, keep inner separators thin
			r[0] = '║'
			r[len(r)-1] = '║'
		}
		out = append(out, string(r))
	}
	return strings.Join(out, "\n")
}

// extractTable reassembles the screen cells written by tview into lines of
// plain text, skipping zero-width spacer cells that follow wide (CJK) runes.
func extractTable(cells []tcell.SimCell, width, height int) string {
	lines := make([][]rune, height)
	for i := range lines {
		lines[i] = make([]rune, 0, width)
	}
	skipNext := false
	for i, c := range cells {
		if skipNext {
			skipNext = false
			continue
		}
		if len(c.Runes) == 0 {
			continue
		}
		r := c.Runes[0]
		y := i / width
		lines[y] = append(lines[y], r)
		if runewidth.RuneWidth(r) >= 2 {
			skipNext = true
		}
	}

	var out []string
	for _, line := range lines {
		s := strings.TrimRight(string(line), " ")
		if s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, "\n")
}
