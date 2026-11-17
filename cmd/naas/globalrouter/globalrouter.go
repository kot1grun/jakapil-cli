/*
Copyright © 2026 Kot1gruN
*/
package globalrouter

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/naas/client"
)

// NewGlobalRouterCommand returns the parent "naas global-router" command with its
// subcommands: list, create, update and delete. It manages Global Router (NaaS)
// routers through the selectel/globalrouter-go V1 API.
func NewGlobalRouterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "global-router",
		Short:   "Manage Global Router routers",
		Long:    `Create, list, update and delete Selectel Global Router (NaaS) routers.`,
		Aliases: []string{"gr"},
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
		newCreateCommand(),
		newUpdateCommand(),
		newDeleteCommand(),
	)

	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Global Router routers",
		Long:  `List the routers of the project, optionally filtered by name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "filter routers by name")
	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Global Router router",
		Long: `Create a new Global Router router.
The name is required; tags are optional (repeat --tags or pass a comma-separated list).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "router name (required)")
	cmd.Flags().StringSlice("tags", nil, "router tags (repeat the flag or pass a comma-separated list)")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <router-id>",
		Short: "Update a Global Router router",
		Long: `Update attributes of an existing router by its ID.

At least one of --name or --tags must be provided.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args[0])
		},
	}
	cmd.Flags().StringP("name", "n", "", "new router name")
	cmd.Flags().StringSlice("tags", nil, "new router tags (repeat the flag or pass a comma-separated list)")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <router-id>",
		Short:   "Delete a Global Router router",
		Long:    `Delete an existing router by its ID.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runDelete(cmd, c, args[0])
		},
	}
	return cmd
}
