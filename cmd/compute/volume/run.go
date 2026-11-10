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

// runCreate creates a volume and renders the created volume.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	opts, err := volumeCreateOpts(cmd)
	if err != nil {
		return err
	}

	volume, err := volumes.Create(cmd.Context(), c, opts, nil).Extract()
	if err != nil {
		return err
	}
	return renderShow(cmd, volume)
}

// volumeUpdateOpts assembles the volumes.UpdateOpts from the update flags,
// returning an error when no update option was requested.
func volumeUpdateOpts(cmd *cobra.Command) (volumes.UpdateOpts, error) {
	nameChanged := cmd.Flags().Changed("name")
	descChanged := cmd.Flags().Changed("description")
	if !nameChanged && !descChanged {
		return volumes.UpdateOpts{}, errors.New("at least one update option is required: --name or --description")
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
	return opts, nil
}

// runUpdate applies at least one update option and renders the updated volume.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts, err := volumeUpdateOpts(cmd)
	if err != nil {
		return err
	}

	volume, err := volumes.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return renderShow(cmd, volume)
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
