package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type sample struct {
	Name  string `json:"name" yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

// runRoot executes a root command with a persistent --output flag and a single
// leaf command returning outValue via Print(cmd, outValue).
func runRoot(t *testing.T, args []string, outValue interface{}) (string, error) {
	t.Helper()

	root := &cobra.Command{Use: "root"}
	root.PersistentFlags().String("output", YAML, "output format")
	root.SetArgs(args)

	leaf := &cobra.Command{
		Use:   "leaf",
		Short: "leaf",
		RunE: func(cmd *cobra.Command, args []string) error {
			return Print(cmd, outValue)
		},
	}
	root.AddCommand(leaf)

	var buf bytes.Buffer
	root.SetOut(&buf)

	err := root.Execute()
	return buf.String(), err
}

func TestPrintYAMLDefault(t *testing.T) {
	out, err := runRoot(t, []string{"leaf"}, sample{Name: "test", Count: 2})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, "name: test") || !strings.Contains(out, "count: 2") {
		t.Fatalf("yaml output mismatch:\n%s", out)
	}
}

func TestPrintJSON(t *testing.T) {
	out, err := runRoot(t, []string{"leaf", "--output", "json"}, sample{Name: "test", Count: 2})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out, `"name": "test"`) || !strings.Contains(out, `"count": 2`) {
		t.Fatalf("json output mismatch:\n%s", out)
	}
}

// embeddedBaseView mimics the mks-go Cluster, which embeds an exported
// BaseView struct holding some fields tagged with json:"-".
type embeddedBaseView struct {
	ID string `json:"id"`
	// Secret is hidden in JSON (json:"-") and must stay hidden in YAML too.
	Secret string `json:"-"`
}

type clusterLike struct {
	Name string `json:"name"`
	embeddedBaseView
}

// TestPrintYAMLMatchesJSON verifies that the YAML renderer produces the same
// field set as the JSON renderer: it must not render anonymous embedded
// structs (e.g. BaseView) as a nested key and must honour json:"-".
func TestPrintYAMLMatchesJSON(t *testing.T) {
	v := clusterLike{Name: "c1", embeddedBaseView: embeddedBaseView{ID: "abc", Secret: "hidden"}}

	yamlOut, err := runRoot(t, []string{"leaf" /* default yaml */}, v)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	jsonOut, err := runRoot(t, []string{"leaf", "--output", "json"}, v)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if !strings.Contains(yamlOut, "name: c1") || !strings.Contains(yamlOut, "id: abc") {
		t.Fatalf("yaml should contain promoted BaseView fields, got:\n%s", yamlOut)
	}
	if strings.Contains(yamlOut, "embeddedBaseView") || strings.Contains(yamlOut, "baseview") {
		t.Fatalf("yaml must not render the embedded struct as a nested key:\n%s", yamlOut)
	}
	if strings.Contains(yamlOut, "secret") {
		t.Fatalf("yaml must not expose json:\"-\" field, got:\n%s\nyaml json:\n%s", yamlOut, jsonOut)
	}
	if strings.Contains(jsonOut, "secret") {
		t.Fatalf("json unexpectedly contains json:\"-\" field:\n%s", jsonOut)
	}
}

func TestPrintPlainList(t *testing.T) {
	out, err := runRoot(t, []string{"leaf", "--output="}, []sample{{Name: "a"}, {Name: "b"}})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if strings.Count(out, "Name:") != 2 {
		t.Fatalf("plain list should print each element, got:\n%s", out)
	}
}

func TestPrintUnknownFormat(t *testing.T) {
	if _, err := runRoot(t, []string{"leaf", "--output", "xml"}, sample{Name: "test"}); err == nil {
		t.Fatal("expected error for unknown output format")
	}
}

func TestPrintTableEmpty(t *testing.T) {
	cmd := &cobra.Command{Use: "leaf"}
	buf := &strings.Builder{}
	cmd.SetOut(buf)

	if err := PrintTable(cmd, []string{"ID", "Name"}, nil); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	if !strings.Contains(buf.String(), "No resources found.") {
		t.Fatalf("expected a plain empty notice, got: %q", buf.String())
	}
	if strings.Contains(buf.String(), "║") {
		t.Fatalf("must not render a header-only table, got: %q", buf.String())
	}
}

func TestRenderTableAlignment(t *testing.T) {
	got, err := RenderTable(
		[]string{"ID", "Name", "Status"},
		[][]string{
			{"net-1", "nf", "ACTIVE"},
			{"net-222", "a-longer-name", "DOWN"},
		},
	)
	if err != nil {
		t.Fatalf("RenderTable: %v", err)
	}

	// The table is a full grid: borders around every cell, a border between
	// each column, and a row separator under the header. The outer frame uses
	// double lines; the internal separators stay thin.
	want := "╔═══════╤═════════════╤══════╗\n" +
		"║ID     │Name         │Status║\n" +
		"╠───────┼─────────────┼──────╣\n" +
		"║net-1  │nf           │ACTIVE║\n" +
		"╠───────┼─────────────┼──────╣\n" +
		"║net-222│a-longer-name│DOWN  ║\n" +
		"╚═══════╧═════════════╧══════╝"
	if got != want {
		t.Fatalf("table output mismatch.\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestIsTable(t *testing.T) {
	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().String("output", Table, "")
	if !IsTable(cmd) {
		t.Fatal("default format must be table")
	}
	if err := cmd.Flags().Set("output", YAML); err != nil {
		t.Fatal(err)
	}
	if IsTable(cmd) {
		t.Fatal("expected IsTable to be false after --output yaml")
	}
}
