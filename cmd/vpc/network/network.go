/*
Copyright © 2026 Kot1gruN
*/
package network

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
)

// NewNetworkCommand returns the parent "vpc network" command with its
// subcommands: list, create, show, update and delete.
func NewNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage VPC networks",
		Long: `Create, list, show, update and delete virtual private cloud (VPC)
networks.`,
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
		Short: "List VPC networks",
		Long:  `List the networks in the project, optionally filtered by name, status or ID.`,
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
	cmd.Flags().StringP("name", "n", "", "filter networks by name")
	cmd.Flags().String("status", "", "filter networks by status (ACTIVE, DOWN, BUILD, ERROR)")
	cmd.Flags().String("id", "", "filter networks by ID")
	cmd.Flags().Int("limit", 0, "maximum number of networks to return")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list networks in every region the service advertises instead of --region")
	return cmd
}

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show <network-id>",
		Short:   "Show a VPC network",
		Long:    `Show the details of a single VPC network by its ID.`,
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
		Short: "Create a VPC network",
		Long: `Create a new VPC network.

The network name is required.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewNetworkServiceClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().StringP("name", "n", "", "network name")
	cmd.Flags().String("description", "", "human-readable network description")
	cmd.Flags().Bool("admin-state-up", true, "set the administrative state up")
	cmd.Flags().Bool("shared", false, "allow the network to be shared with other projects")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <network-id>",
		Short: "Update a VPC network",
		Long: `Update the attributes of an existing VPC network by its ID.

At least one of --name, --description, --admin-state-up or --shared must be
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
	cmd.Flags().StringP("name", "n", "", "new network name")
	cmd.Flags().String("description", "", "new network description")
	cmd.Flags().Bool("admin-state-up", false, "set the administrative state up")
	cmd.Flags().Bool("shared", false, "allow the network to be shared with other projects")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <network-id>",
		Short:   "Delete a VPC network",
		Long:    `Delete a VPC network by its ID.`,
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
