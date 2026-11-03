/*
Copyright © 2026 Kot1gruN
*/
package router

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists routers, applying the --name, --status, --description, --id
// and --limit filters when set, and renders the result through the --output
// flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchRouters(cmd.Context(), c, routersListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runListAllRegions lists routers in every region the service advertises and
// renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only
// GETs. A region whose endpoint is unavailable is skipped with a warning on
// stderr; the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewNetworkServiceClients(cmd)
	if err != nil {
		return err
	}
	opts := routersListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchRouters(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, r := range list {
			rows = append(rows, routerRow(r))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, routerHeader, results)
}

// routersListOpts builds the routers list options from the list flags.
func routersListOpts(cmd *cobra.Command) routers.ListOpts {
	opts := routers.ListOpts{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		opts.Name = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		opts.Status = v
	}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		opts.Description = v
	}
	if v, _ := cmd.Flags().GetString("id"); v != "" {
		opts.ID = v
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	return opts
}

// fetchRouters lists routers through the paginator and returns them flat.
func fetchRouters(ctx context.Context, c *gophercloud.ServiceClient, opts routers.ListOpts) ([]routers.Router, error) {
	var list []routers.Router
	pager := routers.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageRouters, err := routers.ExtractRouters(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageRouters...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// runCreate builds the create options from the flags and renders the created
// router.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	opts := routers.CreateOpts{
		Name:        mustString(cmd, "name"),
		Description: mustString(cmd, "description"),
	}
	if cmd.Flags().Changed("admin-state-up") {
		val, _ := cmd.Flags().GetBool("admin-state-up")
		opts.AdminStateUp = &val
	}
	if cmd.Flags().Changed("distributed") {
		val, _ := cmd.Flags().GetBool("distributed")
		opts.Distributed = &val
	}
	if v, _ := cmd.Flags().GetString("gateway-network-id"); v != "" {
		opts.GatewayInfo = &routers.GatewayInfo{NetworkID: v}
	}

	router, err := routers.Create(cmd.Context(), c, opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, router)
}

// runUpdate applies at least one update option and renders the updated router.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts := routers.UpdateOpts{}
	applied := false

	if cmd.Flags().Changed("name") {
		opts.Name = mustString(cmd, "name")
		applied = true
	}
	if cmd.Flags().Changed("description") {
		val := mustString(cmd, "description")
		opts.Description = &val
		applied = true
	}
	if cmd.Flags().Changed("admin-state-up") {
		val, _ := cmd.Flags().GetBool("admin-state-up")
		opts.AdminStateUp = &val
		applied = true
	}
	if cmd.Flags().Changed("distributed") {
		val, _ := cmd.Flags().GetBool("distributed")
		opts.Distributed = &val
		applied = true
	}
	if cmd.Flags().Changed("gateway-network-id") {
		val := mustString(cmd, "gateway-network-id")
		opts.GatewayInfo = &routers.GatewayInfo{NetworkID: val}
		applied = true
	}
	if !applied {
		return fmt.Errorf("at least one update option is required: --name, --description, --admin-state-up, --distributed or --gateway-network-id")
	}

	router, err := routers.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, router)
}

// runDelete removes a router by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := routers.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "router %s deleted\n", args[0])
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// runPortAdd attaches an internal interface to a router by subnet or port ID
// and prints a confirmation line. AddInterfaceOpts requires exactly one of
// subnet_id or port_id, mirrored here as "--subnet-id or --port-id".
func runPortAdd(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	routerID := args[0]
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	portID, _ := cmd.Flags().GetString("port-id")

	if (subnetID == "") == (portID == "") {
		return fmt.Errorf("exactly one of --subnet-id or --port-id is required")
	}
	opts := routers.AddInterfaceOpts{SubnetID: subnetID, PortID: portID}

	info, err := routers.AddInterface(cmd.Context(), c, routerID, opts).Extract()
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "attached interface to router %s (port %s, subnet %s)\n", routerID, info.PortID, info.SubnetID)
	return nil
}

// runPortDelete detaches an internal interface from a router by subnet or port
// ID and prints a confirmation line. RemoveInterfaceOpts requires at least one
// of subnet_id or port_id, mirrored here as "--subnet-id or --port-id".
func runPortDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	routerID := args[0]
	subnetID, _ := cmd.Flags().GetString("subnet-id")
	portID, _ := cmd.Flags().GetString("port-id")

	if subnetID == "" && portID == "" {
		return fmt.Errorf("at least one of --subnet-id or --port-id is required")
	}
	opts := routers.RemoveInterfaceOpts{SubnetID: subnetID, PortID: portID}

	info, err := routers.RemoveInterface(cmd.Context(), c, routerID, opts).Extract()
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "detached interface from router %s (port %s, subnet %s)\n", routerID, info.PortID, info.SubnetID)
	return nil
}
