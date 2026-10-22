/*
Copyright © 2026 Kot1gruN
*/
package craas

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
	"jakapil-cli/cmd/craas/registry"
	"jakapil-cli/cmd/mks/output"
)

// NewCraasCommand returns the parent "craas" command for the Selectel
// Container Registry Service.
func NewCraasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "craas",
		Short: "Manage Selectel Container Registry Service (CRaaS)",
		Long: `Manage Selectel Container Registry Service (CRaaS).

The following objects are supported: registries, repositories`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Persistent flags are shared by all craas subcommands.
	cmd.PersistentFlags().StringP(client.FlagToken, "t", "", "Selectel API token (default $SLT_KS_TOKEN)")
	cmd.PersistentFlags().String(client.FlagEndpoint, client.DefaultEndpoint, "CRaaS API endpoint (default $SLT_CR_ENDPOINT)")
	cmd.PersistentFlags().StringP("output", "o", output.YAML, "output format: json or yaml")

	cmd.AddCommand(
		registry.NewRegistryCommand(),
	)

	return cmd
}
