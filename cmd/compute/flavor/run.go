/*
Copyright © 2026 Kot1gruN
*/
package flavor

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists flavors and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchFlavors(cmd.Context(), c, flavorsListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runListAllRegions lists flavors in every region the service advertises and
// renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only GETs.
// A region whose endpoint is unavailable is skipped with a warning on stderr;
// the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewComputeServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := flavorsListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchFlavors(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, f := range list {
			rows = append(rows, flavorRow(f))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, flavorHeader, results)
}

// flavorsListOpts builds the flavors list options from the list flags.
func flavorsListOpts(cmd *cobra.Command) flavors.ListOpts {
	opts := flavors.ListOpts{}
	if v, _ := cmd.Flags().GetString("changes-since"); v != "" {
		opts.ChangesSince = v
	}
	if v, _ := cmd.Flags().GetInt("min-disk"); v > 0 {
		opts.MinDisk = v
	}
	if v, _ := cmd.Flags().GetInt("min-ram"); v > 0 {
		opts.MinRAM = v
	}
	if v, _ := cmd.Flags().GetString("sort-dir"); v != "" {
		opts.SortDir = v
	}
	if v, _ := cmd.Flags().GetString("sort-key"); v != "" {
		opts.SortKey = v
	}
	if v, _ := cmd.Flags().GetString("is-public"); v != "" {
		opts.AccessType = flavors.AccessType(v)
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	return opts
}

// fetchFlavors lists flavors through the paginator and returns them flat.
func fetchFlavors(ctx context.Context, c *gophercloud.ServiceClient, opts flavors.ListOpts) ([]flavors.Flavor, error) {
	var list []flavors.Flavor
	pager := flavors.ListDetail(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageFlavors, err := flavors.ExtractFlavors(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageFlavors...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// runCreate builds the create options from the flags and renders the created
// flavor.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	name, _ := cmd.Flags().GetString("name")
	ram, _ := cmd.Flags().GetInt("ram")
	vcpus, _ := cmd.Flags().GetInt("vcpus")
	disk, _ := cmd.Flags().GetInt("disk")
	if name == "" {
		return fmt.Errorf("flavor name is required: pass --name")
	}
	if ram <= 0 || vcpus <= 0 || disk <= 0 {
		return fmt.Errorf("--ram, --vcpus and --disk must be greater than zero")
	}

	opts := flavors.CreateOpts{
		Name:  name,
		RAM:   ram,
		VCPUs: vcpus,
		Disk:  &disk,
	}
	if v, _ := cmd.Flags().GetString("id"); v != "" {
		opts.ID = v
	}
	if v, _ := cmd.Flags().GetInt("swap"); v > 0 {
		swap := v
		opts.Swap = &swap
	}
	if v, _ := cmd.Flags().GetFloat64("rxtx-factor"); v != 0 {
		opts.RxTxFactor = v
	}
	if cmd.Flags().Changed("is-public") {
		val, _ := cmd.Flags().GetBool("is-public")
		opts.IsPublic = &val
	}
	if v, _ := cmd.Flags().GetInt("ephemeral"); v > 0 {
		ephemeral := v
		opts.Ephemeral = &ephemeral
	}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		opts.Description = v
	}

	flavor, err := flavors.Create(cmd.Context(), c, opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, flavor)
}

// runUpdate applies at least one update option and renders the updated flavor.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if !cmd.Flags().Changed("description") {
		return fmt.Errorf("at least one update option is required: --description")
	}

	opts := flavors.UpdateOpts{
		Description: mustString(cmd, "description"),
	}

	flavor, err := flavors.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, flavor)
}

// runDelete removes a flavor by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := flavors.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "flavor %s deleted\n", args[0])
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
