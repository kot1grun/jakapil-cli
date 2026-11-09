/*
Copyright © 2026 Kot1gruN
*/
package image

import (
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
)

// NewImageCommand returns the parent "compute image" command with its
// subcommands: list, create, update and delete.
func NewImageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manage compute images",
		Long:  `Create, list, update and delete OpenStack Glance (image) images via the image v2 API.`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
		newShowCommand(),
		newCreateCommand(),
		newUploadCommand(),
		newDownloadCommand(),
		newUpdateCommand(),
		newDeleteCommand(),
	)

	return cmd
}

func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <image-id>",
		Short: "Show a compute image",
		Long:  `Show the details of a single image by its ID.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewImageV2ServiceClient(cmd)
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
		Short: "List compute images",
		Long:  `List the images, optionally filtered by name, status, visibility, owner or size.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRegions, _ := cmd.Flags().GetBool(client.FlagAllRegions); allRegions {
				return runListAllRegions(cmd)
			}
			c, err := client.NewImageV2ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runList(cmd, c)
		},
	}
	cmd.Flags().String("name", "", "filter images by name")
	cmd.Flags().String("status", "", "filter images by status, e.g. active, queued, saving (or in:status1,status2)")
	cmd.Flags().String("visibility", "", "filter by visibility: public, private, shared or community")
	cmd.Flags().String("owner", "", "filter by the project ID that owns the images")
	cmd.Flags().Int("limit", 0, "maximum number of images to return")
	cmd.Flags().String("sort", "", "sort results, e.g. name:asc,size:desc (new-style sort)")
	cmd.Flags().StringArray("tag", nil, "filter images carrying any of the given tags (repeatable)")
	cmd.Flags().Int64("min-size", 0, "filter images with size at least this many bytes")
	cmd.Flags().Int64("max-size", 0, "filter images with size at most this many bytes")
	cmd.Flags().String("disk-format", "", "filter by disk format, e.g. qcow2, raw, iso")
	cmd.Flags().String("container-format", "", "filter by container format, e.g. bare, ovf")
	cmd.Flags().Bool("hidden", false, "only list hidden images (os_hidden)")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list images in every region the service advertises instead of --region")
	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an image",
		Long: `Create a new image record.

The name (--name), container format (--container-format) and disk format
(--disk-format) are required. Pass --file to upload the local image payload
into Glance right after the record is created, turning it into an "active"
image in a single step.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewImageV2ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runCreate(cmd, c)
		},
	}
	cmd.Flags().String("name", "", "image name (required)")
	cmd.Flags().String("id", "", "image UUID (defaults to an auto-generated one)")
	cmd.Flags().String("container-format", "", "container format: bare, ovf, ami, ari or aki (required)")
	cmd.Flags().String("disk-format", "", "disk format: qcow2, raw, iso, vhd, vmdk, vdi, ami, ari or aki (required)")
	cmd.Flags().StringP("file", "f", "", "path to a local image file to upload into Glance")
	cmd.Flags().Bool("no-progress", false, "do not render a progress bar while uploading")
	cmd.Flags().String("visibility", "", "image visibility: public or private")
	cmd.Flags().Bool("hidden", false, "hide the image from the default image list (os_hidden)")
	cmd.Flags().Int("min-disk", 0, "minimum disk space in GB required to boot the image")
	cmd.Flags().Int("min-ram", 0, "minimum RAM in MB required to boot the image")
	cmd.Flags().Bool("protected", false, "prevent the image from being deleted")
	cmd.Flags().StringArray("tag", nil, "tag to attach to the image (repeatable)")
	cmd.Flags().StringToString("property", nil, "custom property as key=value (repeatable)")
	return cmd
}

// newUploadCommand returns the "image upload" subcommand that streams a local
// file into an existing image. Glance accepts data only when the image is in the
// "queued" state, so upload deals with images produced by create --no-file (or
// left behind by a failed create --file without wiping the record).
func newUploadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload <image-id>",
		Short: "Upload image data to a queued image",
		Long: `Upload the raw image payload into an existing image record.

The image must be in the "queued" status (for example, one created without
--file, or left over from an interrupted create --file). After a successful
upload the image transitions to "active" automatically.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewImageV2ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runUpload(cmd, c, args)
		},
	}
	cmd.Flags().StringP("file", "f", "", "path to the local image file to upload (required)")
	cmd.Flags().Bool("no-progress", false, "do not render a progress bar while uploading")
	return cmd
}

// newDownloadCommand returns the "image download" subcommand that streams an
// image's raw payload into a local file (--file) or stdout. A progress bar
// (determinate when the server reports Content-Length, otherwise an
// indeterminate spinner) is rendered on stderr.
func newDownloadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <image-id>",
		Short: "Download image data to a file",
		Long: `Stream the raw image payload into a local file (or stdout when --file is
omitted).

Download progress is shown on stderr, so redirecting stdout to a file (e.g.
via shell redirection) does not corrupt the output.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewImageV2ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runDownload(cmd, c, args)
		},
	}
	cmd.Flags().StringP("file", "f", "", "path to save the image payload to (default: stdout)")
	cmd.Flags().Bool("force", false, "overwrite the target file if it already exists")
	cmd.Flags().Bool("no-progress", false, "do not render a progress bar while downloading")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <image-id>",
		Short: "Update an image",
		Long:  `Update attributes of an existing image by its ID. At least one update option is required.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewImageV2ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runUpdate(cmd, c, args)
		},
	}
	cmd.Flags().String("name", "", "new image name")
	cmd.Flags().String("visibility", "", "new visibility: public, private, shared or community")
	cmd.Flags().Bool("protected", false, "prevent or allow deletion of the image")
	cmd.Flags().Bool("hidden", false, "show or hide the image in the default image list")
	cmd.Flags().Int("min-disk", 0, "new minimum disk space in GB")
	cmd.Flags().Int("min-ram", 0, "new minimum RAM in MB")
	cmd.Flags().StringArray("tag", nil, "replace the image tags with the given ones (repeatable)")
	cmd.Flags().StringToString("property", nil, "add or replace a custom property as key=value (repeatable)")
	cmd.Flags().StringArray("remove-property", nil, "remove a custom property by key (repeatable)")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <image-id>",
		Short:   "Delete an image",
		Long:    `Delete an image by its ID.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.NewImageV2ServiceClient(cmd)
			if err != nil {
				return err
			}
			return runDelete(cmd, c, args)
		},
	}
	return cmd
}
