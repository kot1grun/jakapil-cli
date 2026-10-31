/*
Copyright © 2026 Kot1gruN
*/
package vpc

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
	"jakapil-cli/cmd/vpc/network"
	"jakapil-cli/cmd/vpc/port"
	"jakapil-cli/cmd/vpc/subnet"
	"jakapil-cli/internal/output"
)

// NewVpcCommand returns the parent "vpc" command for the Selectel Virtual
// Private Cloud (VPC) networking service.
func NewVpcCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpc",
		Short: "Manage Selectel Virtual Private Cloud (VPC)",
		Long: `Manage Selectel Virtual Private Cloud (VPC) networking.

The following objects are supported: networks and subnets.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Persistent flags are shared by all vpc subcommands.
	cmd.PersistentFlags().StringP(client.FlagToken, "t", "", "Selectel Keystone API token (default $SLT_KS_TOKEN)")
	cmd.PersistentFlags().StringP(client.FlagEndpoint, "e", client.DefaultEndpoint, "Selectel Keystone Identity v3 auth URL for authentication and service discovery (default $OS_AUTH_URL)")
	cmd.PersistentFlags().String(client.FlagRegion, "", "target region for VPC endpoint discovery, e.g. ru-1 (default $OS_REGION_NAME)")
	cmd.PersistentFlags().Bool(client.FlagDebug, false, "print the discovered service catalog and VPC endpoint to stderr")
	cmd.PersistentFlags().StringP("output", "o", output.Table, "output format: table, json or yaml")

	cmd.AddCommand(
		network.NewNetworkCommand(),
		subnet.NewSubnetCommand(),
		port.NewPortCommand(),
	)

	return cmd
}
