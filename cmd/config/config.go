/*
Copyright © 2026 Kot1gruN
*/
package config

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"jakapil-cli/internal/authentication"
)

const (
	// flagEnvPrefix selects the environment variable prefix inspected by the
	// config list command (default "OS_", like openrc exports).
	flagEnvPrefix = "env-prefix"

	// flagConfigFile points at the jakapil-cli.yaml to display. It defaults to
	// the file in the configuration directory.
	flagConfigFile = "config-file"

	// defaultEnvPrefix is used when --env-prefix is not given.
	defaultEnvPrefix = "OS_"

	// headers for the two rendered sections.
	sectionConfig = "Configuration file"
	sectionEnv    = "Environment variables"
)

// recognizedSuffixes lists the OpenStack environment variable names (relative
// to the chosen prefix, e.g. "AUTH_URL") that are consulted by the gophercloud
// clientconfig package this CLI builds its credentials from. Variables outside
// this list are not taken into account.
var recognizedSuffixes = []string{
	"APPLICATION_CREDENTIAL_ID",
	"APPLICATION_CREDENTIAL_NAME",
	"APPLICATION_CREDENTIAL_SECRET",
	"AUTH_TOKEN",
	"AUTH_URL",
	"CACERT",
	"CERT",
	"CLOUD",
	"DEFAULT_DOMAIN",
	"DOMAIN_ID",
	"DOMAIN_NAME",
	"IDENTITY_API_VERSION",
	"INSECURE",
	"KEY",
	"PASSWORD",
	"PROJECT_DOMAIN_ID",
	"PROJECT_DOMAIN_NAME",
	"PROJECT_ID",
	"PROJECT_NAME",
	"REGION_NAME",
	"SYSTEM_SCOPE",
	"TENANT_ID",
	"TENANT_NAME",
	"TOKEN",
	"TRUST_ID",
	"USERNAME",
	"USER_DOMAIN_ID",
	"USER_DOMAIN_NAME",
	"USER_ID",
}

// NewConfigCommand returns the "config" command. It shows the help when invoked
// without a subcommand and provides the "list" subcommand.
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect the jakapil-cli configuration",
		Long: `Inspect the jakapil-cli configuration.

The following subcommands are available:
  list    show the on-disk configuration file and the environment variables
          that are taken into account`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(newListCommand())

	return cmd
}

// newListCommand returns the "config list" command. It prints the contents of
// jakapil-cli.yaml (the contexts produced by "onboard") and the environment
// variables with the chosen prefix that the CLI actually takes into account.
func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show the configuration file and the recognized environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd)
		},
	}

	cmd.Flags().String(flagEnvPrefix, defaultEnvPrefix, "environment variable prefix to inspect (default \"OS_\")")
	cmd.Flags().String(flagConfigFile, "", "path to jakapil-cli.yaml to display (default is the file inside the configuration directory)")

	return cmd
}

// runList renders the two sections: the configuration file contents and the
// recognized environment variables set for the chosen prefix.
func runList(cmd *cobra.Command) error {
	prefix, _ := cmd.Flags().GetString(flagEnvPrefix)
	configFile, _ := cmd.Flags().GetString(flagConfigFile)

	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "%s:\n", sectionConfig)
	if err := printConfigFile(out, configFile); err != nil {
		return err
	}

	fmt.Fprintf(out, "\n%s (prefix %q):\n", sectionEnv, prefix)
	renderEnv(out, prefix)

	return nil
}

// printConfigFile writes the raw contents of the configuration file. When no
// explicit path is given the default location is used; a missing file produces
// a short notice instead of an error.
func printConfigFile(w io.Writer, configFile string) error {
	path := configFile
	if path == "" {
		var err error
		path, err = authentication.ConfigFilePath()
		if err != nil {
			return fmt.Errorf("failed to resolve the configuration file path: %w", err)
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(w, "  (no configuration file at %s)\n", path)
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	fmt.Fprintln(w, "  file:", path)
	fmt.Fprintln(w, "  contents:")
	for _, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		fmt.Fprintf(w, "    %s\n", line)
	}
	return nil
}

// renderEnv prints every recognized environment variable for prefix, sorting the
// output by name. Secret-looking values (passwords, tokens, credential secrets)
// are masked so the dump does not leak credentials.
func renderEnv(w io.Writer, prefix string) {
	names := make([]string, 0, len(recognizedSuffixes))
	for _, s := range recognizedSuffixes {
		names = append(names, prefix+s)
	}
	sort.Strings(names)

	for _, name := range names {
		value, ok := os.LookupEnv(name)
		if !ok {
			fmt.Fprintf(w, "  %-40s (unset)\n", name)
			continue
		}

		display := value
		if isSecret(name) {
			display = "*****"
		}
		fmt.Fprintf(w, "  %-40s = %s\n", name, display)
	}
}

// isSecret reports whether an environment variable name refers to a credential
// that should be masked in the output.
func isSecret(name string) bool {
	upper := strings.ToUpper(name)
	for _, marker := range []string{"PASSWORD", "TOKEN", "SECRET", "APPLICATION_CREDENTIAL"} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}
