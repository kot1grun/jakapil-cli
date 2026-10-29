/*
Copyright © 2026 Kot1gruN
*/
package subnet

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
)

// NewSubnetCommand returns the parent "vpc subnet" command with its
// subcommands: list, create, show, update and delete.
func NewSubnetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subnet",
		Short: "Manage VPC subnets",
		Long: `Create, list, show, update and delete virtual private cloud (VPC)
subnets.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
		newCreateCommand(),
		newShowCommand(),
		newUpdateCommand(),
		newDeleteCommand(),
	)

	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VPC subnets",
		Long:  `List the subnets in the project, optionally filtered by network, CIDR, name or ID.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "filter subnets by name")
	cmd.Flags().String("network-id", "", "filter subnets by parent network ID")
	cmd.Flags().String("cidr", "", "filter subnets by CIDR")
	cmd.Flags().String("id", "", "filter subnets by ID")
	cmd.Flags().Int("ip-version", 0, "filter subnets by IP version (4 or 6)")
	cmd.Flags().Int("limit", 0, "maximum number of subnets to return")
	return cmd
}

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <subnet-id>",
		Short:   "Show a VPC subnet",
		Long:    `Show the details of a single VPC subnet by its ID.`,
		Aliases: []string{"get"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runShow(cmd, c, args)
		},
	}
	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VPC subnet",
		Long: `Create a new VPC subnet.

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
	cmd.Flags().String("cidr", "", "address CIDR of the subnet")
	cmd.Flags().StringP("name", "n", "", "subnet name")
	cmd.Flags().String("description", "", "human-readable subnet description")
	cmd.Flags().Int("ip-version", 4, "IP version of the subnet (4 or 6)")
	cmd.Flags().Bool("enable-dhcp", true, "enable the DHCP service")
	cmd.Flags().String("gateway-ip", "", "gateway IP address")
	cmd.Flags().StringSlice("dns-nameservers", nil, "DNS nameservers served via DHCP")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <subnet-id>",
		Short: "Update a VPC subnet",
		Long: `Update the attributes of an existing VPC subnet by its ID.

At least one of --name, --description, --enable-dhcp or --gateway-ip must be
provided.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args)
		},
	}
	cmd.Flags().StringP("name", "n", "", "new subnet name")
	cmd.Flags().String("description", "", "new subnet description")
	cmd.Flags().Bool("enable-dhcp", false, "enable the DHCP service")
	cmd.Flags().String("gateway-ip", "", "gateway IP address")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <subnet-id>",
		Short:   "Delete a VPC subnet",
		Long:    `Delete a VPC subnet by its ID.`,
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
