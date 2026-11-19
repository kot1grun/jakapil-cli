/*
Copyright © 2026 Kot1gruN
*/
package publicsubnet

import (
	"github.com/spf13/cobra"
)

// NewPublicSubnetCommand returns the parent "public-subnet" command with its
// subcommands: list, create and delete. Public subnets are managed by the
// Selectel Resell v2 API (https://docs.selectel.ru/api/cloud-projects-and-resources/#tag/Subnets),
// which accepts a project-scoped token — the same default scope used by the
// MKS/CRaaS commands and by the regular "vpc subnet" command.
//
// The SDK does not expose an update operation for public subnets, so the
// command intentionally omits "update".
func NewPublicSubnetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "public-subnet",
		Short: "Manage VPC public subnets (Resell v2)",
		Long: `Create, list and delete public subnets through the Selectel Resell v2 API.

Every call uses a project-scoped token, obtained with
"jakapil-cli authenticate token issue" (the default scope) and normally
persisted by that command for reuse.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
		newCreateCommand(),
		newDeleteCommand(),
	)

	return cmd
}
