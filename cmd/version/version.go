/*
Copyright © 2026 Kot1gruN
*/
package version

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"jakapil-cli/internal/logo"
)

// Build-time variables, injected by the linker flags (-ldflags "-X ..."). When
// the binary is built without them (e.g. plain `go build`), the zero values
// below are kept.
var (
	// version is the most recent git tag the build was made from. It falls back
	// to "0.0.0" when no tag is present (the original repo has no tags yet).
	version = "0.0.0"

	// gitCommit is the short hash of the commit the binary was built from.
	gitCommit = ""

	// buildDate is the UTC build timestamp (rendered in RFC3339).
	buildDate = ""
)

// NewVersionCommand returns the "version" command. It prints the build identity
// of the running binary: version, git commit, build date and the Go runtime the
// binary was compiled with.
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the jakapil-cli version and build information",
		Long: `Print the jakapil-cli version and build information.

The fields Version, GitCommit and BuildDate are stamped at build time via
linker flags (see the Makefile). GoVersion, Compiler and Platform always come
from the Go runtime that actually built the binary.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			printVersion(cmd)
			return nil
		},
	}
}

// printVersion renders the version block to the command output.
func printVersion(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s\n\n", logo.Banner)
	fmt.Fprintf(out, "Version:   %s\n", version)
	fmt.Fprintf(out, "GitCommit: %s\n", gitCommit)
	fmt.Fprintf(out, "BuildDate: %s\n", buildDate)
	fmt.Fprintf(out, "GoVersion: %s\n", runtime.Version())
	fmt.Fprintf(out, "Compiler:  %s\n", runtime.Compiler)
	fmt.Fprintf(out, "Platform:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
