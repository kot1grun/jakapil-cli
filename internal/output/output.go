/*
Copyright © 2026 Kot1gruN
*/
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

const (
	// JSON represents the JSON output format.
	JSON = "json"

	// YAML represents the YAML output format.
	YAML = "yaml"
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
