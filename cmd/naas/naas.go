/*
Copyright © 2026 Kot1gruN
*/
package naas

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/naas/client"
	"jakapil-cli/cmd/naas/globalrouter"
	"jakapil-cli/cmd/naas/services"
	"jakapil-cli/cmd/naas/zonegroups"
	"jakapil-cli/cmd/naas/zones"
	"jakapil-cli/internal/output"
)

// NewNaasCommand returns the parent "naas" command for the Selectel Network as a
// Service (NaaS) product family. Its subcommands manage the individual NaaS
// services; the API endpoint is fixed (https://api.selectel.ru/naas/v1), so no
// Keystone service discovery is involved.
func NewNaasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "naas",
		Short: "Manage Selectel Network as a Service (NaaS)",
		Long: `Manage Selectel Network as a Service (NaaS) networking.

The API endpoint is fixed (https://api.selectel.ru/naas/v1) and does not require
Keystone service discovery: only an authentication token is needed.

The following objects are supported: global-router, zones, zonegroups, services.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Persistent flags are shared by all naas subcommands.
	cmd.PersistentFlags().StringP(client.FlagToken, "t", "", "Selectel Keystone API token, must be account-scoped (default $SLT_KS_TOKEN)")
	cmd.PersistentFlags().StringP("output", "o", output.Table, "output format: table|dt|json|yaml")

	cmd.AddCommand(
		globalrouter.NewGlobalRouterCommand(),
		zones.NewZonesCommand(),
		zonegroups.NewZoneGroupsCommand(),
		services.NewServicesCommand(),
	)

	return cmd
}
