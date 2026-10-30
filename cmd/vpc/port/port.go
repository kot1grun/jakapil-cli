/*
Copyright © 2026 Kot1gruN
*/
package port

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
)

// NewPortCommand returns the parent "vpc port" command with its subcommands:
// list, create, update and delete.
func NewPortCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "port",
		Short: "Manage VPC ports",
		Long:  `Create, list, update and delete virtual private cloud (VPC) ports.`,
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
		Short: "List VPC ports",
		Long:  `List the ports in the project, optionally filtered by network, status, name, MAC address or ID.`,
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
	cmd.Flags().StringP("name", "n", "", "filter ports by name")
	cmd.Flags().String("network-id", "", "filter ports by network ID")
	cmd.Flags().String("status", "", "filter ports by status (ACTIVE, DOWN, BUILD, ERROR)")
	cmd.Flags().String("mac-address", "", "filter ports by MAC address")
	cmd.Flags().String("device-id", "", "filter ports by attached device ID")
	cmd.Flags().String("id", "", "filter ports by ID")
	cmd.Flags().Int("limit", 0, "maximum number of ports to return")
	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VPC port",
		Long: `Create a new VPC port.

The parent network is required (--network-id).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("network-id", "", "ID of the parent network (required)")
	cmd.Flags().StringP("name", "n", "", "port name")
	cmd.Flags().String("description", "", "human-readable port description")
	cmd.Flags().Bool("admin-state-up", true, "set the administrative state up")
	cmd.Flags().String("mac-address", "", "MAC address for the port")
	cmd.Flags().String("device-id", "", "ID of the device attached to this port")
	cmd.Flags().String("device-owner", "", "the type of device that owns this port")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <port-id>",
		Short: "Update a VPC port",
		Long: `Update the attributes of an existing VPC port by its ID.

At least one of --name, --description, --admin-state-up, --device-id or
--device-owner must be provided.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args)
		},
	}
	cmd.Flags().StringP("name", "n", "", "new port name")
	cmd.Flags().String("description", "", "new port description")
	cmd.Flags().Bool("admin-state-up", false, "set the administrative state up")
	cmd.Flags().String("device-id", "", "ID of the device attached to this port")
	cmd.Flags().String("device-owner", "", "the type of device that owns this port")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <port-id>",
		Short:   "Delete a VPC port",
		Long:    `Delete a VPC port by its ID.`,
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
