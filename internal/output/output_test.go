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

func TestIsUUID(t *testing.T) {
	valid := []string{
		"11111111-1111-1111-1111-111111111111",
		"88888888-8888-8888-8888-888888888888",
		"abcdef01-1234-5678-9abc-def012345678",
	}
	invalid := []string{
		"",
		"cloud_net",
		"11111111-1111-1111-1111-11111111111",   // 35 chars
		"11111111-1111-1111-1111-1111111111111", // 37 chars
		"11111111-zzzz-1111-1111-111111111111",  // non-hex group
		"11111111-1111-111x-1111-111111111111",  // non-hex digit
		"111111111111-1111-1111-111111111111",   // dash in wrong position
	}
	for _, s := range valid {
		if !isUUID(s) {
			t.Fatalf("expected isUUID(%q) = true", s)
		}
	}
	for _, s := range invalid {
		if isUUID(s) {
			t.Fatalf("expected isUUID(%q) = false", s)
		}
	}
}

func TestWrapDenseCells(t *testing.T) {
	uuid := "11111111-1111-1111-1111-111111111111"
	rows := [][]string{
		{uuid, "cloud_net", "ACTIVE", "prod,ha,blue"},
	}
	got := wrapDenseCells(rows)
	// Original slice is not mutated.
	if a, b := rows[0][0], uuid; a != b {
		t.Fatalf("input row must be unchanged, got %q", rows[0][0])
	}
	if want := "11111111-\n1111-\n1111-\n1111-\n111111111111"; got[0][0] != want {
		t.Fatalf("uuid cell not wrapped.\nwant: %q\ngot:  %q", want, got[0][0])
	}
	if got[0][1] != "cloud_net" || got[0][2] != "ACTIVE" {
		t.Fatalf("non-uuid value cells must be untouched: %v", got[0])
	}
	if want := "prod\nha\nblue"; got[0][3] != want {
		t.Fatalf("comma list not split.\nwant: %q\ngot:  %q", want, got[0][3])
	}
}

func TestSplitListTrimsSpaces(t *testing.T) {
	got := splitList(" prod ,ha ,  blue ")
	if want := "prod\nha\nblue"; got != want {
		t.Fatalf("splitList mismatch.\nwant: %q\ngot:  %q", want, got)
	}
}

func TestPrintTableWrapsUUID(t *testing.T) {
	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", Table, "")
	buf := &strings.Builder{}
	cmd.SetOut(buf)

	uuid := "11111111-1111-1111-1111-111111111111"
	_ = cmd.Flags().Set("output", DenseTable)
	if err := PrintTable(cmd, []string{"ID", "Name"}, [][]string{{uuid, "cloud_net"}}); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	if !strings.Contains(buf.String(), "1111-") {
		t.Fatalf("expected hyphen to end a wrapped line, got:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), uuid) {
		t.Fatalf("full unwrapped uuid must not appear, got:\n%s", buf.String())
	}
}

func TestPrintTableDefaultKeepsUUID(t *testing.T) {
	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", Table, "")
	buf := &strings.Builder{}
	cmd.SetOut(buf)

	uuid := "11111111-1111-1111-1111-111111111111"
	if err := PrintTable(cmd, []string{"ID", "Name"}, [][]string{{uuid, "cloud_net"}}); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	if !strings.Contains(buf.String(), uuid) {
		t.Fatalf("default table must keep the uuid on one line, got:\n%s", buf.String())
	}
}

func TestPrintTableWrapsTags(t *testing.T) {
	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", Table, "")
	buf := &strings.Builder{}
	cmd.SetOut(buf)

	_ = cmd.Flags().Set("output", DenseTable)
	if err := PrintTable(cmd, []string{"Name", "Tags"}, [][]string{{"cloud_net", "prod,ha,blue"}}); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	if !strings.Contains(buf.String(), "prod") || !strings.Contains(buf.String(), "ha") || !strings.Contains(buf.String(), "blue") {
		t.Fatalf("dense table must show each tag on its own line, got:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "prod,ha,blue") {
		t.Fatalf("dense table must not keep the comma-joined tag list, got:\n%s", buf.String())
	}
}

func TestPrintTableDefaultKeepsTags(t *testing.T) {
	cmd := &cobra.Command{Use: "leaf"}
	cmd.Flags().StringP("output", "o", Table, "")
	buf := &strings.Builder{}
	cmd.SetOut(buf)

	if err := PrintTable(cmd, []string{"Name", "Tags"}, [][]string{{"cloud_net", "prod,ha,blue"}}); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	if !strings.Contains(buf.String(), "prod,ha,blue") {
		t.Fatalf("default table must keep tags comma-joined on one line, got:\n%s", buf.String())
	}
}
