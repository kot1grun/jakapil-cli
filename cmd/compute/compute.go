/*
Copyright © 2026 Kot1gruN
*/
package compute

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
	"jakapil-cli/cmd/compute/flavor"
	"jakapil-cli/cmd/compute/image"
	"jakapil-cli/internal/output"
)

// NewComputeCommand returns the parent "compute" command for the Selectel
// Compute (OpenStack Nova) service.
func NewComputeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compute",
		Short: "Manage Selectel Compute (OpenStack Nova) resources",
		Long: `Manage Selectel Compute (OpenStack Nova) resources.

The following objects are supported: flavors, images.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Persistent flags are shared by all compute subcommands.
	cmd.PersistentFlags().StringP(client.FlagToken, "t", "", "Selectel Keystone API token (default $SLT_KS_TOKEN)")
	cmd.PersistentFlags().StringP(client.FlagEndpoint, "e", client.DefaultEndpoint, "Selectel Keystone Identity v3 auth URL for authentication and service discovery (default $OS_AUTH_URL)")
	cmd.PersistentFlags().String(client.FlagRegion, "", "target region for compute endpoint discovery, e.g. ru-1 (default $OS_REGION_NAME)")
	cmd.PersistentFlags().Bool(client.FlagDebug, false, "print the discovered service catalog and compute endpoint to stderr")
	cmd.PersistentFlags().StringP("output", "o", output.Table, "output format: table, json or yaml")

	cmd.AddCommand(
		flavor.NewFlavorCommand(),
		image.NewImageCommand(),
	)

	return cmd
}
