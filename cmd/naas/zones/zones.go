/*
Copyright © 2026 Kot1gruN
*/
package zones

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/naas/client"
)

// NewZonesCommand returns the parent "naas zones" command with its subcommands.
// It manages NaaS Global Router zones through the selectel/globalrouter-go V1
// API.
func NewZonesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zones",
		Short: "Manage Global Router zones",
		Long:  `List the Global Router (NaaS) zones available to the project.`,
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
		Short: "List Global Router zones",
		Long:  `List the Global Router zones available to the project, optionally filtered.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "filter zones by name")
	cmd.Flags().String("service-id", "", "filter zones by service ID")
	cmd.Flags().Bool("enable", false, "filter enabled zones only")
	return cmd
}
