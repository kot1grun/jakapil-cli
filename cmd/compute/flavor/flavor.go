/*
Copyright © 2026 Kot1gruN
*/
package flavor

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
)

// NewFlavorCommand returns the parent "compute flavor" command with its
// subcommands: list, create, update and delete.
func NewFlavorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flavor",
		Short: "Manage compute flavors",
		Long:  `Create, list, show, update and delete OpenStack Nova (compute) flavors.`,
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
		Use:   "show <flavor-id>",
		Short: "Show a compute flavor",
		Long:  `Show the details of a single compute flavor by its ID.`,
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
		Short: "List compute flavors",
		Long:  `List the flavors in the project, optionally filtered by disk, RAM or access scope.`,
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
	cmd.Flags().String("changes-since", "", "filter flavors changed since the given ISO 8601 timestamp")
	cmd.Flags().Int("min-disk", 0, "filter flavors with at least this much root disk (GB)")
	cmd.Flags().Int("min-ram", 0, "filter flavors with at least this much RAM (MB)")
	cmd.Flags().String("sort-dir", "", "sort direction: asc or desc")
	cmd.Flags().String("sort-key", "", "sort by this flavor attribute (default flavorid)")
	cmd.Flags().String("is-public", "", "filter by visibility: true (public+project), false (private, admin only) or None (all, admin only)")
	cmd.Flags().Int("limit", 0, "maximum number of flavors to return")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list flavors in every region the service advertises instead of --region")
	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a compute flavor",
		Long: `Create a new compute flavor.

The name (--name), RAM (--ram, MB), vCPUs (--vcpus) and root disk
(--disk, GB) are required.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewComputeServiceClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("name", "", "flavor name (required)")
	cmd.Flags().Int("ram", 0, "flavor memory in MB (required)")
	cmd.Flags().Int("vcpus", 0, "number of virtual CPUs (required)")
	cmd.Flags().Int("disk", 0, "amount of root disk in GB (required)")
	cmd.Flags().String("id", "", "unique ID for the flavor (defaults to an auto-generated one)")
	cmd.Flags().Int("swap", 0, "amount of swap space in MB")
	cmd.Flags().Float64("rxtx-factor", 0, "network bandwidth factor for the flavor")
	cmd.Flags().Int("ephemeral", 0, "amount of ephemeral disk in GB")
	cmd.Flags().String("description", "", "free-form flavor description")
	cmd.Flags().Bool("highfreq", false, "request a high-frequency (highfreq) flavor profile")
	cmd.Flags().Bool("dedicated-vcpus", false, "dedicate physical CPU cores to the flavor")
	// Disabled for now: the active API microversion does not accept a
	// vcpus_vendor property on flavor create. Re-enable together with the
	// payload wiring in run.go once the API supports it.
	// cmd.Flags().String("vcpus-vendor", "", "CPU vendor for the flavor, e.g. amd")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <flavor-id>",
		Short: "Update a compute flavor",
		Long: `Update the description of an existing compute flavor by its ID.

Currently the Nova API supports updating only the flavor description.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewComputeServiceClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args)
		},
	}
	cmd.Flags().String("description", "", "new flavor description (required)")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <flavor-id>",
		Short:   "Delete a compute flavor",
		Long:    `Delete a compute flavor by its ID.`,
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
	return cmd
}
