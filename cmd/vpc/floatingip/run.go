/*
Copyright © 2026 Kot1gruN
*/
package floatingip

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists floating IPs, applying the --id, --description,
// --floating-network-id, --port-id, --fixed-ip-address, --floating-ip-address
// and --status filters when set, and renders the result through the --output
// flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchFloatingIPs(cmd.Context(), c, floatingIPListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runListAllRegions lists floating IPs in every region the service advertises
// and renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only GETs.
// A region whose endpoint is unavailable is skipped with a warning on stderr;
// the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewNetworkServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := floatingIPListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchFloatingIPs(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, f := range list {
			rows = append(rows, floatingIPRow(f))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, floatingIPHeader, results)
}

// floatingIPListOpts builds the floating IP list options from the list flags.
func floatingIPListOpts(cmd *cobra.Command) floatingips.ListOpts {
	opts := floatingips.ListOpts{}
	if v, _ := cmd.Flags().GetString("id"); v != "" {
		opts.ID = v
	}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		opts.Description = v
	}
	if v, _ := cmd.Flags().GetString("floating-network-id"); v != "" {
		opts.FloatingNetworkID = v
	}
	if v, _ := cmd.Flags().GetString("port-id"); v != "" {
		opts.PortID = v
	}
	if v, _ := cmd.Flags().GetString("fixed-ip-address"); v != "" {
		opts.FixedIP = v
	}
	if v, _ := cmd.Flags().GetString("floating-ip-address"); v != "" {
		opts.FloatingIP = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		opts.Status = v
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	return opts
}

// fetchFloatingIPs lists floating IPs through the paginator and returns them
// flat.
func fetchFloatingIPs(ctx context.Context, c *gophercloud.ServiceClient, opts floatingips.ListOpts) ([]floatingips.FloatingIP, error) {
	var list []floatingips.FloatingIP
	pager := floatingips.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageFIPs, err := floatingips.ExtractFloatingIPs(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageFIPs...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// runShow renders a single floating IP selected by its ID.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	floatingIP, err := floatingips.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	return render(cmd, floatingIP)
}

// runCreate builds the create options from the flags and renders the created
// floating IP.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	networkID, _ := cmd.Flags().GetString("floating-network-id")
	if networkID == "" {
		return fmt.Errorf("external network is required: pass --floating-network-id")
	}

	opts := floatingips.CreateOpts{
		FloatingNetworkID: networkID,
		Description:       mustString(cmd, "description"),
		FloatingIP:        mustString(cmd, "floating-ip-address"),
		PortID:            mustString(cmd, "port-id"),
		FixedIP:           mustString(cmd, "fixed-ip-address"),
		SubnetID:          mustString(cmd, "subnet-id"),
	}

	floatingIP, err := floatingips.Create(cmd.Context(), c, opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, floatingIP)
}

// runUpdate applies at least one update option and renders the updated
// floating IP.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts := floatingips.UpdateOpts{}
	applied := false

	if cmd.Flags().Changed("port-id") {
		val, _ := cmd.Flags().GetString("port-id")
		opts.PortID = &val
		applied = true
	}
	if cmd.Flags().Changed("description") {
		val, _ := cmd.Flags().GetString("description")
		opts.Description = &val
		applied = true
	}
	if cmd.Flags().Changed("fixed-ip-address") {
		val, _ := cmd.Flags().GetString("fixed-ip-address")
		opts.FixedIP = val
		applied = true
	}
	if !applied {
		return fmt.Errorf("at least one update option is required: --port-id, --description or --fixed-ip-address")
	}

	floatingIP, err := floatingips.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, floatingIP)
}

// runDelete removes a floating IP by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := floatingips.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "floating IP %s deleted\n", args[0])
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
