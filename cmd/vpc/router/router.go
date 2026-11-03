/*
Copyright © 2026 Kot1gruN
*/
package router

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
)

// NewRouterCommand returns the parent "vpc router" command with its
// subcommands: list, create, update and delete. It manages Logical L3 routers
// through the gophercloud layer3/routers extension.
func NewRouterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "router",
		Short: "Manage VPC routers",
		Long:  `Create, list, update and delete virtual private cloud (VPC) L3 routers.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
		newCreateCommand(),
		newUpdateCommand(),
		newDeleteCommand(),
		newPortCommand(),
	)

	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VPC routers",
		Long:  `List the routers in the project, optionally filtered by name, status, description or ID.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRegions, _ := cmd.Flags().GetBool(client.FlagAllRegions); allRegions {
				return runListAllRegions(cmd)
			}
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "filter routers by name")
	cmd.Flags().String("status", "", "filter routers by status (ACTIVE, BUILD, ERROR)")
	cmd.Flags().String("description", "", "filter routers by description")
	cmd.Flags().String("id", "", "filter routers by ID")
	cmd.Flags().Int("limit", 0, "maximum number of routers to return")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list routers in every region the service advertises instead of --region")
	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VPC router",
		Long: `Create a new VPC L3 router.

The external gateway is optional (--gateway-network-id).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "router name")
	cmd.Flags().String("description", "", "human-readable router description")
	cmd.Flags().Bool("admin-state-up", true, "set the administrative state up")
	cmd.Flags().Bool("distributed", false, "create a distributed router")
	cmd.Flags().String("gateway-network-id", "", "ID of the external network to use as gateway")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <router-id>",
		Short: "Update a VPC router",
		Long: `Update the attributes of an existing VPC router by its ID.

At least one of --name, --description, --admin-state-up, --distributed or
--gateway-network-id must be provided.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args)
		},
	}
	cmd.Flags().StringP("name", "n", "", "new router name")
	cmd.Flags().String("description", "", "new router description")
	cmd.Flags().Bool("admin-state-up", false, "set the administrative state up")
	cmd.Flags().Bool("distributed", false, "set the router as distributed")
	cmd.Flags().String("gateway-network-id", "", "ID of the external network to use as gateway")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <router-id>",
		Short:   "Delete a VPC router",
		Long:    `Delete a VPC router by its ID.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runDelete(cmd, c, args)
		},
	}
	return cmd
}

func newPortCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "port",
		Short: "Manage router interfaces",
		Long:  `Attach (add) and detach (delete) internal router interfaces to subnets or ports.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newPortAddCommand(),
		newPortDeleteCommand(),
	)

	return cmd
}

func newPortAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <router-id>",
		Short: "Attach an interface to a VPC router",
		Long: `Attach an internal router interface to a subnet or a port.

Exactly one of --subnet-id or --port-id is required.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runPortAdd(cmd, c, args)
		},
	}
	cmd.Flags().String("subnet-id", "", "ID of the subnet to attach")
	cmd.Flags().String("port-id", "", "ID of the port to attach")
	return cmd
}

func newPortDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <router-id>",
		Short: "Detach an interface from a VPC router",
		Long: `Detach an internal router interface from a subnet or a port.

At least one of --subnet-id or --port-id is required.`,
		Aliases: []string{"rm", "remove"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runPortDelete(cmd, c, args)
		},
	}
	cmd.Flags().String("subnet-id", "", "ID of the subnet to detach")
	cmd.Flags().String("port-id", "", "ID of the port to detach")
	return cmd
}
