/*
Copyright © 2026 Kot1gruN
*/
package network

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/naas/client"
)

// NewNetworkCommand returns the parent "naas global-router network" command with
// its subcommands: create, list, show, update and delete. It manages Global
// Router (NaaS) networks (VPC and dedicated) through the
// selectel/globalrouter-go V1 API.
func NewNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network",
		Short:   "Manage Global Router networks",
		Long:    `Create, list, show, update and disconnect Selectel Global Router (NaaS) networks.`,
		Aliases: []string{"net"},
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newCreateCommand(),
		newListCommand(),
		newShowCommand(),
		newUpdateCommand(),
		newDeleteCommand(),
	)

	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Global Router network",
		Long: `Create a new network on a Global Router router.

Supported types (--type):
  vpc       (default) attaches an OpenStack network, requires --os-network-id,
            --project-id is optional
  dedicated provisions a dedicated VLAN, requires --vlan, --inner-vlan is optional

router-id and zone-id are required for both types; name and tags are optional
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
	cmd.Flags().String("router-id", "", "ID of the router to attach the network to (required)")
	cmd.Flags().String("zone-id", "", "ID of the zone the network belongs to (required)")
	cmd.Flags().String("type", "vpc", "network type: vpc or dedicated")
	cmd.Flags().String("os-network-id", "", "ID of the OpenStack network (VPC networks, required)")
	cmd.Flags().String("project-id", "", "project ID of the network (VPC)")
	cmd.Flags().Int("vlan", 0, "VLAN ID (dedicated networks, required)")
	cmd.Flags().Int("inner-vlan", 0, "inner VLAN ID (dedicated networks)")
	cmd.Flags().StringP("name", "n", "", "network name")
	cmd.Flags().StringSlice("tags", nil, "network tags (repeat the flag or pass a comma-separated list)")
	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Global Router networks",
		Long: `List the networks of the project.
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
	cmd.Flags().StringP("name", "n", "", "filter networks by name")
	cmd.Flags().String("router-id", "", "filter networks by router ID (client-side)")
	return cmd
}

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <network-id>",
		Short: "Show a Global Router network",
		Long:  `Show the full information about a Global Router network by its ID.`,
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

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <network-id>",
		Short: "Update a Global Router network",
		Long: `Update attributes of an existing network by its ID.

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
	cmd.Flags().StringP("name", "n", "", "new network name")
	cmd.Flags().StringSlice("tags", nil, "new network tags (repeat the flag or pass a comma-separated list)")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <network-id>",
		Short:   "Disconnect a Global Router network",
		Long:    `Disconnect an existing network from a router by its ID.`,
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
