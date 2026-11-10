/*
Copyright © 2026 Kot1gruN
*/
package volume

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists volumes and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchVolumes(cmd.Context(), c, volumeListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runShow fetches a single volume by its ID and renders it through the --output
// flag.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	volume, err := volumes.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShow(cmd, volume)
}

// runListAllRegions lists volumes in every region the service advertises and
// renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only GETs.
// A region whose endpoint is unavailable is skipped with a warning on stderr;
// the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewBlockStorageV3ServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := volumeListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchVolumes(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, vol := range list {
			rows = append(rows, volumeRow(vol))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, volumeHeader, results)
}

// volumeListOpts builds the volumes list options from the list flags.
func volumeListOpts(cmd *cobra.Command) volumes.ListOpts {
	opts := volumes.ListOpts{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		opts.Name = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		opts.Status = v
	}
	if v, _ := cmd.Flags().GetString("bootable"); v != "" {
		b := strings.EqualFold(v, "true")
		opts.Bootable = &b
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	if v, _ := cmd.Flags().GetString("sort"); v != "" {
		opts.Sort = v
	}
	return opts
}

// fetchVolumes lists volumes through the paginator and returns them flat.
func fetchVolumes(ctx context.Context, c *gophercloud.ServiceClient, opts volumes.ListOpts) ([]volumes.Volume, error) {
	var list []volumes.Volume
	pager := volumes.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageVolumes, err := volumes.ExtractVolumes(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageVolumes...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// volumeCreateOpts builds the volumes.CreateOpts from the create flags.
func volumeCreateOpts(cmd *cobra.Command) (volumes.CreateOpts, error) {
	name := mustString(cmd, "name")
	if name == "" {
		return volumes.CreateOpts{}, fmt.Errorf("--name is required")
	}
	size, _ := cmd.Flags().GetInt("size")
	if size <= 0 {
		return volumes.CreateOpts{}, fmt.Errorf("--size is required and must be at least 1 GB")
	}

	opts := volumes.CreateOpts{
		Name:             name,
		Size:             size,
		Description:      mustString(cmd, "description"),
		VolumeType:       mustString(cmd, "volume-type"),
		AvailabilityZone: mustString(cmd, "availability-zone"),
		SnapshotID:       mustString(cmd, "snapshot-id"),
		SourceVolID:      mustString(cmd, "source-vol-id"),
		ImageID:          mustString(cmd, "image-ref"),
		BackupID:         mustString(cmd, "backup-id"),
	}
	if md, _ := cmd.Flags().GetStringToString("metadata"); len(md) > 0 {
		opts.Metadata = md
	}
	return opts, nil
}

// runCreate creates a volume (applying the optional bootable/read-only flags
// via post-create actions) and renders the created volume.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	opts, err := volumeCreateOpts(cmd)
	if err != nil {
		return err
	}
	actionOpts, err := parseVolumeActionOpts(cmd)
	if err != nil {
		return err
	}

	volume, err := volumes.Create(cmd.Context(), c, opts, nil).Extract()
	if err != nil {
		return err
	}

	if err := applyVolumeActions(cmd.Context(), c, volume.ID, actionOpts); err != nil {
		return err
	}
	return renderShow(cmd, volume)
}

// volumeUpdateOpts assembles the volumes.UpdateOpts from the update flags,
// returning an error when no update option was requested.
func volumeUpdateOpts(cmd *cobra.Command) (volumes.UpdateOpts, error) {
	nameChanged := cmd.Flags().Changed("name")
	descChanged := cmd.Flags().Changed("description")
	metaChanged := cmd.Flags().Changed("metadata")
	sizeChanged := cmd.Flags().Changed("size")
	if !nameChanged && !descChanged && !metaChanged && !sizeChanged && !hasVolumeFlags(cmd) {
		return volumes.UpdateOpts{}, errors.New("at least one update option is required: --name, --description, --metadata, --size, --bootable, --non-bootable, --read-only or --read-write")
	}

	opts := volumes.UpdateOpts{}
	if nameChanged {
		name := mustString(cmd, "name")
		opts.Name = &name
	}
	if descChanged {
		desc := mustString(cmd, "description")
		opts.Description = &desc
	}
	if metaChanged {
		if md, _ := cmd.Flags().GetStringToString("metadata"); len(md) > 0 {
			opts.Metadata = md
		}
	}
	return opts, nil
}

// volumeExtendSizeOpts returns the requested new size when the --size flag was
// given, or 0 otherwise.
func volumeExtendSizeOpts(cmd *cobra.Command) (newSize int, ok bool, err error) {
	if !cmd.Flags().Changed("size") {
		return 0, false, nil
	}
	size, _ := cmd.Flags().GetInt("size")
	if size <= 0 {
		return 0, false, errors.New("--size must be at least 1 GB")
	}
	return size, true, nil
}

// runUpdate applies at least one update option and renders the updated volume.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts, err := volumeUpdateOpts(cmd)
	if err != nil {
		return err
	}
	actionOpts, err := parseVolumeActionOpts(cmd)
	if err != nil {
		return err
	}
	newSize, sizeSet, err := volumeExtendSizeOpts(cmd)
	if err != nil {
		return err
	}

	// Pull the updateable fields through a PUT only when one of them is set;
	// otherwise (size and/or action-only updates) fetch the current volume so it
	// can be rendered, avoiding a pointless empty PUT.
	var volume *volumes.Volume
	if opts.Name != nil || opts.Description != nil || len(opts.Metadata) > 0 {
		volume, err = volumes.Update(cmd.Context(), c, args[0], opts).Extract()
		if err != nil {
			return err
		}
	} else {
		volume, err = volumes.Get(cmd.Context(), c, args[0]).Extract()
		if err != nil {
			return err
		}
	}

	if sizeSet {
		if err := volumes.ExtendSize(cmd.Context(), c, args[0], volumes.ExtendSizeOpts{NewSize: newSize}).ExtractErr(); err != nil {
			return err
		}
	}

	if err := applyVolumeActions(cmd.Context(), c, args[0], actionOpts); err != nil {
		return err
	}
	return renderShow(cmd, volume)
}

// volumeActionOpts captures whether the bootable and read-only flags request a
// post-create/post-update action and what value to apply.
type volumeActionOpts struct {
	applyBootable bool
	bootable      bool
	applyReadOnly bool
	readOnly      bool
}

// hasVolumeFlags reports whether any bootable/read-only flag was set on the
// command. It is used by update to determine whether an action-only update is
// being requested.
func hasVolumeFlags(cmd *cobra.Command) bool {
	applyBootable, _, _ := bootableFlag(cmd)
	applyReadOnly, _, _ := readOnlyFlag(cmd)
	return applyBootable || applyReadOnly
}

// parseVolumeActionOpts inspects the mutually exclusive bootable/read-only
// flag pairs and returns the actions to apply.
func parseVolumeActionOpts(cmd *cobra.Command) (volumeActionOpts, error) {
	var opts volumeActionOpts

	if ok, bootable, err := bootableFlag(cmd); err != nil {
		return opts, err
	} else if ok {
		opts.applyBootable = true
		opts.bootable = bootable
	}
	if ok, readOnly, err := readOnlyFlag(cmd); err != nil {
		return opts, err
	} else if ok {
		opts.applyReadOnly = true
		opts.readOnly = readOnly
	}
	return opts, nil
}

// bootableFlag resolves the --bootable / --non-bootable pair into (apply, value).
func bootableFlag(cmd *cobra.Command) (bool, bool, error) {
	boot := cmd.Flags().Changed("bootable")
	nonBoot := cmd.Flags().Changed("non-bootable")
	if boot && nonBoot {
		return false, false, errors.New("--bootable and --non-bootable are mutually exclusive")
	}
	if boot {
		v, _ := cmd.Flags().GetBool("bootable")
		return true, v, nil
	}
	if nonBoot {
		return true, false, nil
	}
	return false, false, nil
}

// readOnlyFlag resolves the --read-only / --read-write pair into (apply, readonly).
func readOnlyFlag(cmd *cobra.Command) (bool, bool, error) {
	ro := cmd.Flags().Changed("read-only")
	rw := cmd.Flags().Changed("read-write")
	if ro && rw {
		return false, false, errors.New("--read-only and --read-write are mutually exclusive")
	}
	if ro {
		v, _ := cmd.Flags().GetBool("read-only")
		return true, v, nil
	}
	if rw {
		v, _ := cmd.Flags().GetBool("read-write")
		return true, !v, nil
	}
	return false, false, nil
}

// applyVolumeActions runs the requested bootable and read-only actions against
// the given volume.
func applyVolumeActions(ctx context.Context, c *gophercloud.ServiceClient, id string, opts volumeActionOpts) error {
	if opts.applyBootable {
		if err := volumes.SetBootable(ctx, c, id, volumes.BootableOpts{Bootable: opts.bootable}).ExtractErr(); err != nil {
			return err
		}
	}
	if opts.applyReadOnly {
		if err := setVolumeReadOnly(ctx, c, id, opts.readOnly); err != nil {
			return err
		}
	}
	return nil
}

// setVolumeReadOnly posts the os-update_readonly_flag action to set or clear the
// read-only attribute on a volume. gophercloud does not expose this action
// directly, so it is called via the raw block storage client.
func setVolumeReadOnly(ctx context.Context, c *gophercloud.ServiceClient, id string, readOnly bool) error {
	body := map[string]any{
		"os-update_readonly_flag": map[string]any{"readonly": readOnly},
	}
	resp, err := c.Post(ctx, c.ServiceURL("volumes", id, "action"), body, nil, &gophercloud.RequestOpts{OkCodes: []int{200}})
	_, _, err = gophercloud.ParseResponse(resp, err)
	return err
}

// runDelete removes a volume by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	cascade, _ := cmd.Flags().GetBool("cascade")
	opts := volumes.DeleteOpts{Cascade: cascade}
	if err := volumes.Delete(cmd.Context(), c, args[0], opts).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "volume %s deleted\n", args[0])
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
