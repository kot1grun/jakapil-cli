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
