/*
Copyright © 2026 Kot1gruN
*/
package subnet

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/naas/client"
)

// NewSubnetCommand returns the parent "naas global-router subnet" command with
// its subcommands: list, show, create, update and delete. It manages Global
// Router (NaaS) subnets through the selectel/globalrouter-go V1 API.
func NewSubnetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subnet",
		Short:   "Manage Global Router subnets",
		Long:    `Create, list, show, update and delete Selectel Global Router (NaaS) subnets.`,
		Aliases: []string{"s"},
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
		Short: "List Global Router subnets",
		Long:  `List the subnets of the project, optionally filtered by name.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "filter subnets by name")
	return cmd
}

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <subnet-id>",
		Short: "Show a Global Router subnet",
		Long:  `Show the full information about a Global Router subnet by its ID.`,
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
		Short: "Create a Global Router subnet",
		Long: `Create a new subnet on a Global Router network.
network-id and cidr are required. When --os-subnet-id is set the subnet is created
with the VPC flow (VPCSubnetCreate), otherwise the dedicated flow
(DedicatedSubnetCreate) is used. name, gateway, project-id, service-addresses and
tags are optional (repeat --tags/--service-addresses or pass comma-separated lists).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("network-id", "", "ID of the network to attach the subnet to (required)")
	cmd.Flags().String("cidr", "", "CIDR of the subnet (required)")
	cmd.Flags().String("gateway", "", "gateway IP address of the subnet")
	cmd.Flags().String("os-subnet-id", "", "OpenStack subnet ID (selects the VPC create flow)")
	cmd.Flags().StringP("name", "n", "", "subnet name")
	cmd.Flags().String("project-id", "", "project ID of the subnet (VPC flow only)")
	cmd.Flags().StringSlice("service-addresses", nil, "subnet service addresses (repeat the flag or pass a comma-separated list)")
	cmd.Flags().StringSlice("tags", nil, "subnet tags (repeat the flag or pass a comma-separated list)")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <subnet-id>",
		Short: "Update a Global Router subnet",
		Long: `Update attributes of an existing subnet by its ID.

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
	cmd.Flags().StringP("name", "n", "", "new subnet name")
	cmd.Flags().StringSlice("tags", nil, "new subnet tags (repeat the flag or pass a comma-separated list)")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <subnet-id>",
		Short:   "Delete a Global Router subnet",
		Long:    `Delete an existing subnet by its ID (disconnects it from the router).`,
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
