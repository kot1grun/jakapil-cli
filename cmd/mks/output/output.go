/*
Copyright © 2026 Kot1gruN
*/
package output

import (
	"encoding/json"
	"fmt"

	"go.yaml.in/yaml/v3"
)

const (
	// JSON represents the JSON output format.
	JSON = "json"

	// YAML represents the YAML output format.
	YAML = "yaml"
)

// Render prints the given value to stdout in the requested format.
// An empty format prints nothing and returns false, letting the caller
// fall back to its default representation. Any other format returns an error.
func Render(format string, v interface{}) (bool, error) {
	switch format {
	case "":
		return false, nil
	case JSON:
		return true, renderJSON(v)
	case YAML:
		return true, renderYAML(v)
	default:
		return false, fmt.Errorf("unsupported output format %q: use %q or %q", format, JSON, YAML)
	}
}

func renderJSON(v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func renderYAML(v interface{}) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	return nil
}
