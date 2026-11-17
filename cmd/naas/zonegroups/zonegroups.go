/*
Copyright © 2026 Kot1gruN
*/
package zonegroups

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/naas/client"
)

// NewZoneGroupsCommand returns the parent "naas zonegroups" command with its
// subcommands. It manages NaaS Global Router zone groups through the
// selectel/globalrouter-go V1 API.
func NewZoneGroupsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zonegroups",
		Short: "Manage Global Router zone groups",
		Long:  `List the Global Router (NaaS) zone groups.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
	)

	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Global Router zone groups",
		Long:  `List the Global Router zone groups, optionally filtered by name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "filter zone groups by name")
	return cmd
}
