/*
Copyright © 2026 Kot1gruN
*/
package floatingip

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
)

// NewFloatingIPCommand returns the parent "vpc floating-ip" command with its
// subcommands: list, show, create, update and delete.
func NewFloatingIPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "floating-ip",
		Aliases: []string{"floatingip", "fip"},
		Short:   "Manage VPC floating IPs",
		Long: `Create, list, show, update and delete virtual private cloud (VPC)
floating IP addresses.`,
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
		Short: "List VPC floating IPs",
		Long:  `List the floating IPs in the project, optionally filtered by network, port, address, status or ID.`,
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
	cmd.Flags().String("id", "", "filter floating IPs by ID")
	cmd.Flags().String("description", "", "filter floating IPs by description")
	cmd.Flags().String("floating-network-id", "", "filter floating IPs by external network ID")
	cmd.Flags().String("port-id", "", "filter floating IPs by associated port ID")
	cmd.Flags().String("fixed-ip-address", "", "filter floating IPs by fixed IP address")
	cmd.Flags().String("floating-ip-address", "", "filter floating IPs by floating IP address")
	cmd.Flags().String("status", "", "filter floating IPs by status (ACTIVE, DOWN, ERROR)")
	cmd.Flags().Int("limit", 0, "maximum number of floating IPs to return")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list floating IPs in every region the service advertises instead of --region")
	return cmd
}

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <floating-ip-id>",
		Short:   "Show a VPC floating IP",
		Long:    `Show the details of a single VPC floating IP by its ID.`,
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
		Short: "Create a VPC floating IP",
		Long: `Create a new VPC floating IP.

The external network is required (--floating-network-id). Optionally associate
the floating IP with an internal port (--port-id).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("floating-network-id", "", "ID of the external network where the floating IP is created (required)")
	cmd.Flags().String("description", "", "human-readable floating IP description")
	cmd.Flags().String("floating-ip-address", "", "specific floating IP address to allocate")
	cmd.Flags().String("port-id", "", "ID of the internal port to associate the floating IP with")
	cmd.Flags().String("fixed-ip-address", "", "fixed IP address of the associated port")
	cmd.Flags().String("subnet-id", "", "ID of the subnet from which the floating IP is allocated")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <floating-ip-id>",
		Short: "Update a VPC floating IP",
		Long: `Update the attributes of an existing VPC floating IP by its ID.

At least one of --port-id, --description or --fixed-ip-address must be
provided. Pass --port-id "" to disassociate the floating IP from its port.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args)
		},
	}
	cmd.Flags().String("port-id", "", "new port ID to associate the floating IP with (empty string disassociates)")
	cmd.Flags().String("description", "", "new floating IP description")
	cmd.Flags().String("fixed-ip-address", "", "new fixed IP address of the associated port")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <floating-ip-id>",
		Short:   "Delete a VPC floating IP",
		Long:    `Delete a VPC floating IP by its ID.`,
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
