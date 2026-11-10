/*
Copyright © 2026 Kot1gruN
*/
package volume

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
)

// newTypeCommand returns the parent "compute volume type" command with its
// read-only subcommands: list and show. Volume types are managed by the cloud
// provider, so users cannot create, update or delete them.
func newTypeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "type",
		Short: "Manage compute volume types",
		Long:  `List and show OpenStack Cinder (block storage) volume types. Volume types are provided by Selectel and cannot be created, updated or deleted by users.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newTypeListCommand(),
		newTypeShowCommand(),
	)

	return cmd
}

func newTypeShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <volume-type-id>",
		Short: "Show a compute volume type",
		Long:  `Show the details of a single volume type by its ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewBlockStorageV3ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runTypeShow(cmd, c, args)
		},
	}
	return cmd
}

func newTypeListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List compute volume types",
		Long:  `List the volume types available in the project, optionally filtered by name, description or visibility.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRegions, _ := cmd.Flags().GetBool(client.FlagAllRegions); allRegions {
				return runTypeListAllRegions(cmd)
			}
			c, err := client.NewBlockStorageV3ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runTypeList(cmd, c)
		},
	}
	cmd.Flags().String("name", "", "filter volume types by name")
	cmd.Flags().String("description", "", "filter volume types by description")
	cmd.Flags().String("visibility", "", "filter by visibility: public, private or None (both, default)")
	cmd.Flags().Int("limit", 0, "maximum number of volume types to return")
	cmd.Flags().String("sort", "", "sort results, e.g. name:asc,created_at:desc")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list volume types in every region the service advertises instead of --region")
	return cmd
}
