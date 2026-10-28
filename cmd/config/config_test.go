package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderEnvRespectsPrefixAndMasking(t *testing.T) {
	t.Setenv("OS_AUTH_URL", "https://identity.example.com/v3")
	t.Setenv("OS_USERNAME", "alice")
	t.Setenv("OS_PASSWORD", "s3cret")
	// Not part of the recognized set: must never appear.
	t.Setenv("OS_CUSTOM_UNKNOWN", "ignored")
	// Unrelated prefix variables must be ignored under OS_.
	t.Setenv("MY_AUTH_URL", "https://other.example.net")

	var buf bytes.Buffer
	renderEnv(&buf, "OS_")
	out := buf.String()

	for _, pair := range []struct{ name, value string }{
		{"OS_AUTH_URL", "https://identity.example.com/v3"},
		{"OS_USERNAME", "alice"},
	} {
		if !strings.Contains(out, pair.name) || !strings.Contains(out, pair.value) {
			t.Errorf("output missing recognized variable %q = %q:\n%s", pair.name, pair.value, out)
		}
	}
	if !strings.Contains(out, "OS_PASSWORD") || strings.Contains(out, "s3cret") {
		t.Errorf("password must be masked in output:\n%s", out)
	}
	if strings.Contains(out, "OS_CUSTOM_UNKNOWN") || strings.Contains(out, "MY_AUTH_URL") {
		t.Errorf("unrecognized or foreign-prefix variables must not appear:\n%s", out)
	}
	if !strings.Contains(out, "OS_PROJECT_NAME") || !strings.Contains(out, "(unset)") {
		t.Errorf("recognized-but-unset variables should be reported as (unset):\n%s", out)
	}
}

func TestRenderEnvCustomPrefix(t *testing.T) {
	t.Setenv("MY_OS_AUTH_URL", "https://identity.example.net/v3")
	var buf bytes.Buffer
	renderEnv(&buf, "MY_OS_")
	out := buf.String()
	if !strings.Contains(out, "MY_OS_AUTH_URL") || !strings.Contains(out, "https://identity.example.net/v3") {
		t.Errorf("custom prefix not honored:\n%s", out)
	}
	// No line may begin with the default "OS_" prefix.
	if strings.Contains(out, "\nOS_") {
		t.Errorf("default prefix should not appear under custom prefix:\n%s", out)
	}
}

func TestIsSecret(t *testing.T) {
	for _, name := range []string{"OS_PASSWORD", "OS_TOKEN", "OS_AUTH_TOKEN", "OS_APPLICATION_CREDENTIAL_SECRET", "OS_APPLICATION_CREDENTIAL_ID"} {
		if !isSecret(name) {
			t.Errorf("isSecret(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"OS_AUTH_URL", "OS_USERNAME", "OS_PROJECT_NAME"} {
		if isSecret(name) {
			t.Errorf("isSecret(%q) = true, want false", name)
		}
	}
}

func TestPrintConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "jakapil-cli.yaml")
	content := "contexts:\n  default:\n    region: ru-3\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := printConfigFile(&buf, path); err != nil {
		t.Fatalf("printConfigFile() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "contexts:") || !strings.Contains(out, "region: ru-3") {
		t.Errorf("config file contents not rendered:\n%s", out)
	}
}

func TestPrintConfigFileMissing(t *testing.T) {
	// A missing file should yield a notice, not an error.
	var buf bytes.Buffer
	if err := printConfigFile(&buf, filepath.Join(t.TempDir(), "nope.yaml")); err != nil {
		t.Fatalf("printConfigFile() on missing file = %v", err)
	}
	if !strings.Contains(buf.String(), "no configuration file") {
		t.Errorf("expected a missing-file notice, got:\n%s", buf.String())
	}
}
