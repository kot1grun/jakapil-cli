/*
Copyright © 2026 Kot1gruN
*/
package publicnet

import (
	"github.com/spf13/cobra"

	pclient "jakapil-cli/cmd/vpc/client"
	"jakapil-cli/cmd/vpc/publicnet/client"
)

// NewPublicNetCommand returns the parent "vpc public-port" command with its
// subcommands: list, show, create, update and delete. It manages public ports
// (direct public IP addresses) through the Selectel Public Net API v1.
func NewPublicNetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "public-port",
		Aliases: []string{"public-net", "publicnet", "public-ports"},
		Short:   "Manage VPC public ports (Public Net)",
		Long: `Manage Selectel Public Net public ports, which represent direct public
IP addresses. Supports listing, showing, creating, updating and deleting
public ports through the Public Net API v1.`,
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
		Short: "List VPC public ports",
		Long:  `List the public ports in the project, optionally filtered by project, network, description or ID.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRegions, _ := cmd.Flags().GetBool(pclient.FlagAllRegions); allRegions {
				return runListAllRegions(cmd)
			}
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().String("project-id", "", "filter public ports by project ID")
	cmd.Flags().String("network-id", "", "filter public ports by network ID")
	cmd.Flags().String("description", "", "filter public ports by description")
	cmd.Flags().Bool(pclient.FlagAllRegions, false, "list public ports in every region the service advertises instead of --region")
	return cmd
}

func newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "show <port-id>",
		Short:   "Show a VPC public port",
		Long:    `Show the details of a single Public Net port by its ID.`,
		Aliases: []string{"get"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runShow(cmd, c, args[0])
		},
	}
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VPC public port",
		Long: `Create a new Public Net port (direct public IP address).

The project ID is required (--project-id), unless the project-scoped token
already carries it, in which case it is inferred automatically.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("project-id", "", "ID of the project the public port belongs to (default: inferred from the token scope)")
	cmd.Flags().String("description", "", "human-readable public port description")
	cmd.Flags().StringSlice("security-group-id", nil, "ID of a security group to attach to the public port (repeatable)")
	cmd.Flags().Bool("admin-state-up", true, "set the administrative state up")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <port-id>",
		Short: "Update a VPC public port",
		Long: `Update the attributes of an existing Public Net port by its ID.

At least one of --description, --security-group-id or --admin-state-up must be
provided. Pass --security-group-id to replace the whole security group set.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args[0])
		},
	}
	cmd.Flags().String("description", "", "new public port description")
	cmd.Flags().StringSlice("security-group-id", nil, "security groups to set on the public port (repeatable, replaces the whole set)")
	cmd.Flags().Bool("admin-state-up", false, "set the administrative state up")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <port-id>",
		Short:   "Delete a VPC public port",
		Long:    `Delete a Public Net port by its ID, optionally forcing the deletion.`,
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
	cmd.Flags().Bool("force", false, "force the deletion even when the port is in use")
	return cmd
}
