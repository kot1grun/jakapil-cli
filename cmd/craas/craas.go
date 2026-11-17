/*
Copyright © 2026 Kot1gruN
*/
package craas

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
	"jakapil-cli/cmd/craas/registry"
	"jakapil-cli/cmd/craas/repository"
	"jakapil-cli/cmd/craas/token"
	"jakapil-cli/internal/output"
)

// NewCraasCommand returns the parent "craas" command for the Selectel
// Container Registry Service.
func NewCraasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "craas",
		Short: "Manage Selectel Container Registry Service (CRaaS)",
		Long: `Manage Selectel Container Registry Service (CRaaS).

The following objects are supported: registries, repositories and tokens.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Persistent flags are shared by all craas subcommands.
	cmd.PersistentFlags().StringP(client.FlagToken, "t", "", "Selectel API token (default $SLT_KS_TOKEN)")
	cmd.PersistentFlags().StringP(client.FlagEndpoint, "e", client.DefaultEndpoint, "Selectel Keystone Identity v3 auth URL for authentication and service discovery (default $OS_AUTH_URL)")
	cmd.PersistentFlags().Bool(client.FlagDebug, false, "print the discovered service catalog and CRaaS endpoints to stderr")
	cmd.PersistentFlags().StringP("output", "o", output.Table, "output format: table|dt|json|yaml")

	cmd.AddCommand(
		registry.NewRegistryCommand(),
		repository.NewRepositoryCommand(),
		token.NewTokenCommand(),
	)

	return cmd
}
