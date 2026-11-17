/*
Copyright © 2026 Kot1gruN
*/
package staticroute

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/naas/client"
)

// NewStaticRouteCommand returns the parent "naas global-router static-route"
// command with its subcommands: list, show, create, update and delete. It
// manages Global Router (NaaS) static routes through the selectel/globalrouter-go
// V1 API.
func NewStaticRouteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "static-route",
		Short:   "Manage Global Router static routes",
		Long:    `Create, list, show, update and delete Selectel Global Router (NaaS) static routes.`,
		Aliases: []string{"sr"},
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
		newShowCommand(),
		newCreateCommand(),
		newUpdateCommand(),
		newDeleteCommand(),
	)

	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Global Router static routes",
		Long: `List the static routes of the project.
--name is sent to the API as a filter, --router-id is applied locally over the
returned set because the list API has no router filter.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "filter static routes by name")
	cmd.Flags().String("router-id", "", "filter static routes by router ID (client-side)")
	return cmd
}

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <static-route-id>",
		Short: "Show a Global Router static route",
		Long:  `Show the full information about a Global Router static route by its ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runShow(cmd, c, args[0])
		},
	}
	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Global Router static route",
		Long: `Create a new static route on a Global Router router.
router-id, next-hop and cidr are required; name, project-id and tags are optional
(repeat --tags or pass a comma-separated list).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("router-id", "", "ID of the router to attach the static route to (required)")
	cmd.Flags().String("next-hop", "", "next hop IP address (required)")
	cmd.Flags().String("cidr", "", "destination CIDR of the static route (required)")
	cmd.Flags().StringP("name", "n", "", "static route name")
	cmd.Flags().String("project-id", "", "project ID of the static route")
	cmd.Flags().StringSlice("tags", nil, "static route tags (repeat the flag or pass a comma-separated list)")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <static-route-id>",
		Short: "Update a Global Router static route",
		Long: `Update attributes of an existing static route by its ID.

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
	cmd.Flags().StringP("name", "n", "", "new static route name")
	cmd.Flags().StringSlice("tags", nil, "new static route tags (repeat the flag or pass a comma-separated list)")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <static-route-id>",
		Short:   "Delete a Global Router static route",
		Long:    `Delete an existing static route by its ID.`,
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
