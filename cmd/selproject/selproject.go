/*
Copyright © 2026 Kot1gruN
*/
package selproject

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/selproject/client"
	"jakapil-cli/internal/output"
)

// NewSelProjectCommand returns the parent "selproject" command that manages
// Selectel cloud platform projects through the Resell v2 API (the same API that
// backs /vpc/resell). Calls require an account-scoped IAM token.
func NewSelProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "selproject",
		Short: "Manage Selectel cloud platform projects",
		Long: `Manage cloud platform projects through the Selectel Resell API.

Projects are account-scoped resources: every call needs an account-scoped token,
obtained with "jakapil-cli authenticate token issue --account-scope" and normally
persisted by that command for reuse.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Persistent flags are shared by all selproject subcommands.
	cmd.PersistentFlags().StringP(client.FlagToken, "t", "", "Selectel Keystone API token, must be account-scoped (default $SLT_KS_TOKEN)")
	cmd.PersistentFlags().StringP(client.FlagEndpoint, "e", client.DefaultEndpoint, "Selectel Keystone Identity v3 auth URL for Resell endpoint discovery (default $OS_AUTH_URL)")
	cmd.PersistentFlags().String(client.RegionFlag, "", "Resell catalog region to use, e.g. ru-1 (default: first public region from the catalog)")
	cmd.PersistentFlags().Bool(client.FlagDebug, false, "print the discovered service catalog resell endpoint to stderr")
	cmd.PersistentFlags().StringP("output", "o", output.Table, "output format: table|dt|json|yaml")

	cmd.AddCommand(
		newListCommand(),
		newCreateCommand(),
		newUpdateCommand(),
		newDeleteCommand(),
	)

	return cmd
}
