/*
Copyright © 2026 Kot1gruN
*/
package subnet

import (
	"github.com/spf13/cobra"
)

// NewSubnetCommand returns the parent "vpc subnet" command. It is a placeholder
// for the subnet management operations that are not implemented yet.
func NewSubnetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "subnet",
		Short: "Manage VPC subnets",
		Long: `Manage VPC subnets.

Not implemented yet: the subnet subcommands are coming soon.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
}
