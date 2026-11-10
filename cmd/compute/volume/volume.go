/*
Copyright © 2026 Kot1gruN
*/
package volume

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
)

// NewVolumeCommand returns the parent "compute volume" command with its
// subcommands: list, show, create, update and delete.
func NewVolumeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage compute volumes",
		Long:  `Create, list, show, update and delete OpenStack Cinder (block storage) volumes, and list/show volume types.`,
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
		newTypeCommand(),
	)

	return cmd
}

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <volume-id>",
		Short: "Show a compute volume",
		Long:  `Show the details of a single volume by its ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewBlockStorageV3ServiceClient(cmd)
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
		Short: "List compute volumes",
		Long:  `List the volumes in the project, optionally filtered by name, status, type, bootable or size.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRegions, _ := cmd.Flags().GetBool(client.FlagAllRegions); allRegions {
				return runListAllRegions(cmd)
			}
			c, err := client.NewBlockStorageV3ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().String("name", "", "filter volumes by name")
	cmd.Flags().String("status", "", "filter volumes by status, e.g. available, in-use, creating, error")
	cmd.Flags().String("bootable", "", "filter by whether the volumes are bootable: true or false")
	cmd.Flags().Int("limit", 0, "maximum number of volumes to return")
	cmd.Flags().String("sort", "", "sort results, e.g. size:asc,name:desc")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list volumes in every region the service advertises instead of --region")
	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a compute volume",
		Long: `Create a new block storage volume.

The name (--name) and size (--size, GB) are required. A volume can optionally
be created from a snapshot (--snapshot-id), from another volume (--source-vol-id),
from an image (--image-ref) or from a backup (--backup-id).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewBlockStorageV3ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("name", "", "volume name (required)")
	cmd.Flags().Int("size", 0, "volume size in GB (required)")
	cmd.Flags().String("description", "", "volume description")
	cmd.Flags().String("volume-type", "", "volume type, e.g. fast.ru-1, basic.ru-1 or universal.ru-1")
	cmd.Flags().String("availability-zone", "", "availability zone for the volume")
	cmd.Flags().String("snapshot-id", "", "ID of the snapshot to create the volume from")
	cmd.Flags().String("source-vol-id", "", "ID of the existing volume to clone from")
	cmd.Flags().String("image-ref", "", "ID of the image to create a bootable volume from")
	cmd.Flags().String("backup-id", "", "ID of the backup to create the volume from")
	cmd.Flags().StringToString("metadata", nil, "metadata key=value pair for the volume (repeatable)")
	cmd.Flags().Bool("bootable", false, "mark the volume as bootable")
	cmd.Flags().Bool("non-bootable", false, "mark the volume as non-bootable")
	cmd.Flags().Bool("read-only", false, "make the volume read-only")
	cmd.Flags().Bool("read-write", true, "make the volume read-write (default)")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <volume-id>",
		Short: "Update a compute volume",
		Long:  `Update the name, description, metadata and/or size of an existing volume by its ID. The size can only be grown; shrinking is not supported. At least one update option is required.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewBlockStorageV3ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args)
		},
	}
	cmd.Flags().String("name", "", "new volume name")
	cmd.Flags().String("description", "", "new volume description")
	cmd.Flags().StringToString("metadata", nil, "metadata key=value pair to set on the volume (repeatable, replaces the whole metadata set)")
	cmd.Flags().Int("size", 0, "new volume size in GB (only grow is supported; shrinking is not allowed)")
	cmd.Flags().Bool("bootable", false, "mark the volume as bootable")
	cmd.Flags().Bool("non-bootable", false, "mark the volume as non-bootable")
	cmd.Flags().Bool("read-only", false, "make the volume read-only")
	cmd.Flags().Bool("read-write", false, "make the volume read-write")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <volume-id>",
		Short:   "Delete a compute volume",
		Long:    `Delete a volume by its ID.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewBlockStorageV3ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runDelete(cmd, c, args)
		},
	}
	cmd.Flags().Bool("cascade", false, "delete all snapshots of the volume as well")
	return cmd
}
