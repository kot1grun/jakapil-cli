/*
Copyright © 2026 Kot1gruN
*/
package volume

import (
	"context"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumetypes"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runTypeList lists volume types and renders the result through the --output
// flag.
func runTypeList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchVolumeTypes(cmd.Context(), c, volumeTypeListOpts(cmd))
	if err != nil {
		return err
	}
	return renderTypes(cmd, list)
}

// runTypeShow fetches a single volume type by its ID and renders it through the
// --output flag.
func runTypeShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	volType, err := volumetypes.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderTypeShow(cmd, volType)
}

// runTypeListAllRegions lists volume types in every region the service
// advertises and renders the aggregated result with a Region column (table) or
// grouped by region (json/yaml), backed by the --all-regions flag. Regions are
// queried in parallel with parallel.Regions, since the underlying calls are
// read-only GETs.
func runTypeListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewBlockStorageV3ServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := volumeTypeListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchVolumeTypes(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, vt := range list {
			rows = append(rows, volumeTypeRow(vt))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, volumeTypeHeader, results)
}

// volumeTypeListOpts builds the volume types list options from the list flags.
func volumeTypeListOpts(cmd *cobra.Command) volumetypes.ListOpts {
	opts := volumetypes.ListOpts{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		opts.Name = v
	}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		opts.Description = v
	}
	switch strings.ToLower(mustString(cmd, "visibility")) {
	case "public":
		opts.IsPublic = volumetypes.VisibilityPublic
	case "private":
		opts.IsPublic = volumetypes.VisibilityPrivate
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	if v, _ := cmd.Flags().GetString("sort"); v != "" {
		opts.Sort = v
	}
	return opts
}

// fetchVolumeTypes lists volume types through the paginator and returns them
// flat.
func fetchVolumeTypes(ctx context.Context, c *gophercloud.ServiceClient, opts volumetypes.ListOpts) ([]volumetypes.VolumeType, error) {
	var list []volumetypes.VolumeType
	pager := volumetypes.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageTypes, err := volumetypes.ExtractVolumeTypes(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageTypes...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}
