/*
Copyright © 2026 Kot1gruN
*/
package subnet

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists subnets, applying the --name, --network-id, --cidr, --id and
// --limit filters when set, and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchSubnets(cmd.Context(), c, subnetsListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runListAllRegions lists subnets in every region the service advertises and
// renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only GETs.
// A region whose endpoint is unavailable is skipped with a warning on stderr;
// the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewNetworkServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := subnetsListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchSubnets(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, s := range list {
			rows = append(rows, subnetRow(s))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, subnetHeader, results)
}

// subnetsListOpts builds the subnets list options from the list flags.
func subnetsListOpts(cmd *cobra.Command) subnets.ListOpts {
	opts := subnets.ListOpts{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		opts.Name = v
	}
	if v, _ := cmd.Flags().GetString("network-id"); v != "" {
		opts.NetworkID = v
	}
	if v, _ := cmd.Flags().GetString("cidr"); v != "" {
		opts.CIDR = v
	}
	if v, _ := cmd.Flags().GetString("id"); v != "" {
		opts.ID = v
	}
	if v, _ := cmd.Flags().GetInt("ip-version"); v > 0 {
		opts.IPVersion = v
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	return opts
}

// fetchSubnets lists subnets through the paginator and returns them flat.
func fetchSubnets(ctx context.Context, c *gophercloud.ServiceClient, opts subnets.ListOpts) ([]subnets.Subnet, error) {
	var list []subnets.Subnet
	pager := subnets.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageSubnets, err := subnets.ExtractSubnets(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageSubnets...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// runShow renders a single subnet selected by its ID.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	subnet, err := subnets.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	return render(cmd, subnet)
}

// runCreate builds the create options from the flags and renders the created
// subnet.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	networkID, _ := cmd.Flags().GetString("network-id")
	if networkID == "" {
		return fmt.Errorf("parent network is required: pass --network-id")
	}

	ipVersion, _ := cmd.Flags().GetInt("ip-version")
	if ipVersion != 4 && ipVersion != 6 {
		return fmt.Errorf("unsupported IP version %d: use 4 or 6", ipVersion)
	}

	opts := subnets.CreateOpts{
		NetworkID:   networkID,
		Name:        mustString(cmd, "name"),
		CIDR:        mustString(cmd, "cidr"),
		Description: mustString(cmd, "description"),
		IPVersion:   gophercloud.IPVersion(ipVersion),
	}
	if cmd.Flags().Changed("enable-dhcp") {
		dhcp, _ := cmd.Flags().GetBool("enable-dhcp")
		opts.EnableDHCP = &dhcp
	}
	if gateway, _ := cmd.Flags().GetString("gateway-ip"); gateway != "" {
		opts.GatewayIP = &gateway
	}
	if ns, _ := cmd.Flags().GetStringSlice("dns-nameservers"); len(ns) > 0 {
		opts.DNSNameservers = ns
	}

	subnet, err := subnets.Create(cmd.Context(), c, opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, subnet)
}

// runUpdate applies at least one update option and renders the updated subnet.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts := subnets.UpdateOpts{}
	applied := false

	if cmd.Flags().Changed("name") {
		val := mustString(cmd, "name")
		opts.Name = &val
		applied = true
	}
	if cmd.Flags().Changed("description") {
		val := mustString(cmd, "description")
		opts.Description = &val
		applied = true
	}
	if cmd.Flags().Changed("enable-dhcp") {
		val, _ := cmd.Flags().GetBool("enable-dhcp")
		opts.EnableDHCP = &val
		applied = true
	}
	if cmd.Flags().Changed("gateway-ip") {
		val := mustString(cmd, "gateway-ip")
		opts.GatewayIP = &val
		applied = true
	}
	if !applied {
		return fmt.Errorf("at least one update option is required: --name, --description, --enable-dhcp or --gateway-ip")
	}

	subnet, err := subnets.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, subnet)
}

// runDelete removes a subnet by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := subnets.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "subnet %s deleted\n", args[0])
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
