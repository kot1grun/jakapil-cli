/*
Copyright © 2026 Kot1gruN
*/
package server

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
)

// NewServerCommand returns the parent "compute vm" command (alias
// "virtual-machine") with its subcommands: list, show, create, update and
// delete. It targets the OpenStack Nova compute v2 API through
// gophercloud's openstack/compute/v2/servers package.
func NewServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "vm",
		Aliases: []string{"virtual-machine", "server"},
		Short:   "Manage compute virtual machines",
		Long: `Create, list, show, update and delete Selectel virtual servers.

A "vm" is a virtual machine backed by a flavor (hardware profile) and, usually,
an image (operating system). Under the hood this maps to the Compute v2
/servers API.`,
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

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <server-id>",
		Short: "Show a compute virtual machine",
		Long:  `Show the details of a single virtual machine by its ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewComputeServiceClient(cmd)
			if err != nil {
				return err
			}
			return runShow(cmd, c, args)
		},
	}
	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List compute virtual machines",
		Long:  `List the virtual machines in the project, optionally filtered by name, status, image, flavor or IP.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRegions, _ := cmd.Flags().GetBool(client.FlagAllRegions); allRegions {
				return runListAllRegions(cmd)
			}
			c, err := client.NewComputeServiceClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().String("changes-since", "", "filter servers changed since the given ISO 8601 timestamp")
	cmd.Flags().String("name", "", "filter servers by name (supports regular expressions)")
	cmd.Flags().String("status", "", "filter servers by status, e.g. ACTIVE, BUILD, ERROR, SHUTOFF")
	cmd.Flags().String("image", "", "filter by the image name or ID")
	cmd.Flags().String("flavor", "", "filter by the flavor name or ID")
	cmd.Flags().String("ip", "", "filter by a regular expression matching the IPv4 address")
	cmd.Flags().String("ip6", "", "filter by a regular expression matching the IPv6 address")
	cmd.Flags().String("host", "", "filter by the name of the host (hypervisor)")
	cmd.Flags().Int("limit", 0, "maximum number of servers to return")
	cmd.Flags().String("marker", "", "UUID of the server to start listing after")
	cmd.Flags().String("tags", "", "filter servers having ALL of the given comma-separated tags")
	cmd.Flags().String("tags-any", "", "filter servers having AT LEAST ONE of the given comma-separated tags")
	cmd.Flags().String("not-tags", "", "filter servers NOT having ALL of the given comma-separated tags")
	cmd.Flags().String("not-tags-any", "", "filter servers NOT having ANY of the given comma-separated tags")
	cmd.Flags().String("availability-zone", "", "filter servers by their availability zone")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list servers in every region the service advertises instead of --region")
	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a compute virtual machine",
		Long: `Create a new virtual machine in Selectel.

The name (--name) and flavor (--flavor-ref) are required; the image (--image-ref)
is required unless the server is booted from a volume. The server is attached to
the networks given with --net/--port, to none (--network none) or to "auto"
networks (--network auto).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewComputeServiceClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("name", "", "server name (required)")
	cmd.Flags().String("flavor-ref", "", "ID or name of the flavor describing the server specs (required)")
	cmd.Flags().String("image-ref", "", "ID of the image to boot from (optional when booting from a volume)")
	cmd.Flags().StringArray("net", nil, "UUID of a network to attach the server to (repeatable)")
	cmd.Flags().StringArray("port", nil, "ID of a neutron port to attach the server to (repeatable)")
	cmd.Flags().String("network", "", "attach to networks automatically: auto or none (alternative to --net/--port)")
	cmd.Flags().StringArray("security-group", nil, "name of a security group to apply (repeatable)")
	cmd.Flags().String("availability-zone", "", "availability zone in which to launch the server")
	cmd.Flags().String("user-data-file", "", "path to a file with user data to pass to the server at launch")
	cmd.Flags().StringToString("metadata", nil, "metadata key=value pair (repeatable)")
	cmd.Flags().String("admin-pass", "", "root password for the server (a random one is generated if unset)")
	cmd.Flags().Bool("config-drive", false, "enable the metadata configuration drive")
	cmd.Flags().Bool("config-drive-off", false, "disable the metadata configuration drive")
	cmd.Flags().String("disk-config", "", "disk partitioning: AUTO or MANUAL")
	cmd.Flags().Int("min-count", 0, "minimum number of servers to create")
	cmd.Flags().Int("max-count", 0, "maximum number of servers to create")
	cmd.Flags().StringArray("tag", nil, "tag for the server (repeatable, requires a recent microversion)")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <server-id>",
		Short: "Update a compute virtual machine",
		Long:  `Update attributes of an existing virtual machine by its ID. At least one update option is required.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewComputeServiceClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args)
		},
	}
	cmd.Flags().String("name", "", "new server name (the hostname is not changed)")
	cmd.Flags().String("access-ipv4", "", "new IPv4 address advertised for the server")
	cmd.Flags().String("access-ipv6", "", "new IPv6 address advertised for the server")
	cmd.Flags().String("hostname", "", "new hostname for the server (requires a recent microversion)")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <server-id>",
		Short:   "Delete a compute virtual machine",
		Long:    `Delete a virtual machine by its ID.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewComputeServiceClient(cmd)
			if err != nil {
				return err
			}
			return runDelete(cmd, c, args)
		},
	}
	cmd.Flags().Bool("force", false, "force delete (os-forceDelete) even if the server is stuck in BUILD or ERROR")
	return cmd
}
