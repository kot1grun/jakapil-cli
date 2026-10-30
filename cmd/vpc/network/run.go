/*
Copyright © 2026 Kot1gruN
*/
package network

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists networks, applying the --name, --status, --id and --limit
// filters when set, and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchNetworks(cmd.Context(), c, networksListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runListAllRegions lists networks in every region the service advertises and
// renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only GETs.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewNetworkServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := networksListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchNetworks(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, fmt.Errorf("region %s: %w", region, err)
		}
		rows := make([][]string, 0, len(list))
		for _, n := range list {
			rows = append(rows, networkRow(n))
		}
		return rows, list, nil
	})

	regionRows := make([]output.RegionRows, 0, len(results))
	grouped := make([]output.RegionValue, 0, len(results))
	for _, res := range results {
		if res.Err != nil {
			return res.Err
		}
		regionRows = append(regionRows, output.RegionRows{Region: res.Region, Rows: res.Rows})
		grouped = append(grouped, output.RegionValue{Region: res.Region, Value: res.Value})
	}
	return output.RenderRegions(cmd, networkHeader, regionRows, grouped)
}

// networksListOpts builds the networks list options from the list flags.
func networksListOpts(cmd *cobra.Command) networks.ListOpts {
	opts := networks.ListOpts{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		opts.Name = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		opts.Status = v
	}
	if v, _ := cmd.Flags().GetString("id"); v != "" {
		opts.ID = v
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	return opts
}

// fetchNetworks lists networks through the paginator and returns them flat.
func fetchNetworks(ctx context.Context, c *gophercloud.ServiceClient, opts networks.ListOpts) ([]networks.Network, error) {
	var list []networks.Network
	pager := networks.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageNetworks, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageNetworks...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// runShow renders a single network selected by its ID.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	network, err := networks.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	return render(cmd, network)
}

// runCreate builds the create options from the flags and renders the created
// network.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return fmt.Errorf("network name is required: pass --name")
	}
	description, _ := cmd.Flags().GetString("description")

	opts := networks.CreateOpts{Name: name, Description: description}
	if cmd.Flags().Changed("admin-state-up") {
		val, _ := cmd.Flags().GetBool("admin-state-up")
		opts.AdminStateUp = &val
	}
	if cmd.Flags().Changed("shared") {
		val, _ := cmd.Flags().GetBool("shared")
		opts.Shared = &val
	}

	network, err := networks.Create(cmd.Context(), c, opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, network)
}

// runUpdate applies at least one update option and renders the updated network.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts := networks.UpdateOpts{}
	applied := false

	if cmd.Flags().Changed("name") {
		val, _ := cmd.Flags().GetString("name")
		opts.Name = &val
		applied = true
	}
	if cmd.Flags().Changed("description") {
		val, _ := cmd.Flags().GetString("description")
		opts.Description = &val
		applied = true
	}
	if cmd.Flags().Changed("admin-state-up") {
		val, _ := cmd.Flags().GetBool("admin-state-up")
		opts.AdminStateUp = &val
		applied = true
	}
	if cmd.Flags().Changed("shared") {
		val, _ := cmd.Flags().GetBool("shared")
		opts.Shared = &val
		applied = true
	}
	if !applied {
		return fmt.Errorf("at least one update option is required: --name, --description, --admin-state-up or --shared")
	}

	network, err := networks.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, network)
}

// runDelete removes a network by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := networks.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "network %s deleted\n", args[0])
	return nil
}
