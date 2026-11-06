/*
Copyright © 2026 Kot1gruN
*/
package image

import (
	"context"
	"fmt"
	"sort"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists images and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchImages(cmd.Context(), c, imageListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runShow fetches a single image by its ID and renders it through the --output
// flag.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	image, err := images.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderImage(cmd, image)
}

// runListAllRegions lists images in every region the service advertises and
// renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only GETs.
// A region whose endpoint is unavailable is skipped with a warning on stderr;
// the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewImageV2ServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := imageListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchImages(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, img := range list {
			rows = append(rows, imageRow(img))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, imageHeader, results)
}

// imageListOpts builds the images list options from the list flags.
func imageListOpts(cmd *cobra.Command) images.ListOpts {
	opts := images.ListOpts{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		opts.Name = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		opts.Status = images.ImageStatus(v)
	}
	if v, _ := cmd.Flags().GetString("visibility"); v != "" {
		opts.Visibility = images.ImageVisibility(v)
	}
	if v, _ := cmd.Flags().GetString("owner"); v != "" {
		opts.Owner = v
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	if v, _ := cmd.Flags().GetString("sort"); v != "" {
		opts.Sort = v
	}
	if v, _ := cmd.Flags().GetStringArray("tag"); len(v) > 0 {
		opts.Tags = v
	}
	if v, _ := cmd.Flags().GetInt64("min-size"); v > 0 {
		opts.SizeMin = v
	}
	if v, _ := cmd.Flags().GetInt64("max-size"); v > 0 {
		opts.SizeMax = v
	}
	if v, _ := cmd.Flags().GetString("disk-format"); v != "" {
		opts.DiskFormat = v
	}
	if v, _ := cmd.Flags().GetString("container-format"); v != "" {
		opts.ContainerFormat = v
	}
	if hidden, _ := cmd.Flags().GetBool("hidden"); hidden {
		opts.Hidden = true
	}
	return opts
}

// fetchImages lists images through the paginator and returns them flat.
// The --limit option caps the total number of returned images: the server
// treats it as a per-page size and, via the "next" link, would otherwise feed
// every remaining page, defeating the user's intent to get only N results.
func fetchImages(ctx context.Context, c *gophercloud.ServiceClient, opts images.ListOpts) ([]images.Image, error) {
	var list []images.Image
	pager := images.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageImages, err := images.ExtractImages(page)
		if err != nil {
			return false, err
		}
		for _, img := range pageImages {
			list = append(list, img)
			// Stop walking the pages as soon as the limit is reached.
			if opts.Limit > 0 && len(list) >= opts.Limit {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		return nil, err
	}
	// Guard against a server that ignores limit and returns more than opts.Limit
	// in a single page.
	if opts.Limit > 0 && len(list) > opts.Limit {
		list = list[:opts.Limit]
	}
	return list, nil
}

// runCreate creates an image and renders the result through the --output flag.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	opts, err := imageCreateOpts(cmd)
	if err != nil {
		return err
	}

	image, err := images.Create(cmd.Context(), c, opts).Extract()
	if err != nil {
		return err
	}
	return renderImage(cmd, image)
}

// imageCreateOpts builds the images.CreateOpts from the create flags and
// validates the required fields.
func imageCreateOpts(cmd *cobra.Command) (images.CreateOpts, error) {
	opts := images.CreateOpts{
		Name:            mustString(cmd, "name"),
		ID:              mustString(cmd, "id"),
		Tags:            mustStringArray(cmd, "tag"),
		ContainerFormat: mustString(cmd, "container-format"),
		DiskFormat:      mustString(cmd, "disk-format"),
		MinDisk:         mustInt(cmd, "min-disk"),
		MinRAM:          mustInt(cmd, "min-ram"),
	}
	if opts.Name == "" {
		return images.CreateOpts{}, fmt.Errorf("image name is required: pass --name")
	}
	if opts.ContainerFormat == "" {
		return images.CreateOpts{}, fmt.Errorf("container format is required: pass --container-format")
	}
	if opts.DiskFormat == "" {
		return images.CreateOpts{}, fmt.Errorf("disk format is required: pass --disk-format")
	}

	if v, _ := cmd.Flags().GetString("visibility"); v != "" {
		vis := images.ImageVisibility(v)
		opts.Visibility = &vis
	}
	if hidden, _ := cmd.Flags().GetBool("hidden"); hidden {
		h := hidden
		opts.Hidden = &h
	}
	if protected, _ := cmd.Flags().GetBool("protected"); protected {
		p := protected
		opts.Protected = &p
	}
	if props, _ := cmd.Flags().GetStringToString("property"); len(props) > 0 {
		opts.Properties = props
	}
	return opts, nil
}

// runUpdate applies the requested image updates and renders the updated image.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts, err := imageUpdateOpts(cmd)
	if err != nil {
		return err
	}

	image, err := images.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return renderImage(cmd, image)
}

// imageUpdateOpts assembles the images.UpdateOpts (a JSON patch document) from
// the update flags, returning an error when no update was requested.
func imageUpdateOpts(cmd *cobra.Command) (images.UpdateOpts, error) {
	var patches images.UpdateOpts

	if cmd.Flags().Changed("name") {
		patches = append(patches, images.ReplaceImageName{NewName: mustString(cmd, "name")})
	}
	if cmd.Flags().Changed("visibility") {
		patches = append(patches, images.UpdateVisibility{
			Visibility: images.ImageVisibility(mustString(cmd, "visibility")),
		})
	}
	if cmd.Flags().Changed("protected") {
		protected, _ := cmd.Flags().GetBool("protected")
		patches = append(patches, images.ReplaceImageProtected{NewProtected: protected})
	}
	if cmd.Flags().Changed("hidden") {
		hidden, _ := cmd.Flags().GetBool("hidden")
		patches = append(patches, images.ReplaceImageHidden{NewHidden: hidden})
	}
	if cmd.Flags().Changed("min-disk") {
		patches = append(patches, images.ReplaceImageMinDisk{NewMinDisk: mustInt(cmd, "min-disk")})
	}
	if cmd.Flags().Changed("min-ram") {
		patches = append(patches, images.ReplaceImageMinRam{NewMinRam: mustInt(cmd, "min-ram")})
	}
	if tags := mustStringArray(cmd, "tag"); len(tags) > 0 {
		patches = append(patches, images.ReplaceImageTags{NewTags: tags})
	}
	if props, _ := cmd.Flags().GetStringToString("property"); len(props) > 0 {
		// Deterministic order: map key iteration is random in Go.
		for _, k := range sortedMapKeys(props) {
			patches = append(patches, images.UpdateImageProperty{
				Op:    images.AddOp,
				Name:  k,
				Value: props[k],
			})
		}
	}
	for _, key := range mustStringArray(cmd, "remove-property") {
		patches = append(patches, images.UpdateImageProperty{
			Op:   images.RemoveOp,
			Name: key,
		})
	}

	if len(patches) == 0 {
		return nil, fmt.Errorf("at least one update option is required: --name, --visibility, --protected, --hidden, --min-disk, --min-ram, --tag, --property or --remove-property")
	}
	return patches, nil
}

// runDelete removes an image by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := images.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "image %s deleted\n", args[0])
	return nil
}

func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func mustInt(cmd *cobra.Command, name string) int {
	v, _ := cmd.Flags().GetInt(name)
	return v
}

func mustStringArray(cmd *cobra.Command, name string) []string {
	v, _ := cmd.Flags().GetStringArray(name)
	return v
}

// sortedMapKeys returns the keys of m in ascending order.
func sortedMapKeys(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
