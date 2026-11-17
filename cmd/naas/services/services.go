/*
Copyright © 2026 Kot1gruN
*/
package services

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/naas/client"
)

// NewServicesCommand returns the parent "naas services" command with its
// subcommands. It manages the NaaS Global Router services available to the
// project through the selectel/globalrouter-go V1 API.
func NewServicesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "Manage Global Router services",
		Long:  `List the Global Router (NaaS) services available to the project.`,
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
		Short: "List Global Router services",
		Long:  `List the Global Router services available to the project, optionally filtered by name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "filter services by name")
	return cmd
}
