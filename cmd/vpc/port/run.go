/*
Copyright © 2026 Kot1gruN
*/
package port

import (
	"context"
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/portsecurity"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists ports, applying the --name, --network-id, --status,
// --mac-address, --device-id and --limit filters when set, and renders the
// result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchPorts(cmd.Context(), c, portsListOpts(cmd))
	if err != nil {
		return err
	}
	if len(list) == 0 && output.IsTable(cmd) {
		return output.PrintListEmpty(cmd, client.EffectiveRegion(cmd, c))
	}
	return render(cmd, list)
}

// runListAllRegions lists ports in every region the service advertises and
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
	opts := portsListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchPorts(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, p := range list {
			rows = append(rows, portRow(p))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, portHeader, results)
}

// portsListOpts builds the ports list options from the list flags.
func portsListOpts(cmd *cobra.Command) ports.ListOpts {
	opts := ports.ListOpts{}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		opts.Name = v
	}
	if v, _ := cmd.Flags().GetString("network-id"); v != "" {
		opts.NetworkID = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		opts.Status = v
	}
	if v, _ := cmd.Flags().GetString("mac-address"); v != "" {
		opts.MACAddress = v
	}
	if v, _ := cmd.Flags().GetString("device-id"); v != "" {
		opts.DeviceID = v
	}
	if v, _ := cmd.Flags().GetString("id"); v != "" {
		opts.ID = v
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	return opts
}

// fetchPorts lists ports through the paginator and returns them flat.
func fetchPorts(ctx context.Context, c *gophercloud.ServiceClient, opts ports.ListOpts) ([]ports.Port, error) {
	var list []ports.Port
	pager := ports.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pagePorts, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		list = append(list, pagePorts...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// runCreate builds the create options from the flags and renders the created
// port.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	networkID, _ := cmd.Flags().GetString("network-id")
	if networkID == "" {
		return fmt.Errorf("parent network is required: pass --network-id")
	}

	opts := ports.CreateOpts{
		NetworkID:   networkID,
		Name:        mustString(cmd, "name"),
		Description: mustString(cmd, "description"),
		MACAddress:  mustString(cmd, "mac-address"),
		DeviceID:    mustString(cmd, "device-id"),
		DeviceOwner: mustString(cmd, "device-owner"),
	}
	if cmd.Flags().Changed("admin-state-up") {
		val, _ := cmd.Flags().GetBool("admin-state-up")
		opts.AdminStateUp = &val
	}
	if pairs := addressPairs(cmd); len(pairs) > 0 {
		opts.AllowedAddressPairs = pairs
	}
	if ips, err := fixedIPs(cmd); err != nil {
		return err
	} else if len(ips) > 0 {
		opts.FixedIPs = ips
	}

	var createOpts ports.CreateOptsBuilder = opts
	if cmd.Flags().Changed("port-security-enabled") {
		val, _ := cmd.Flags().GetBool("port-security-enabled")
		createOpts = portsecurity.PortCreateOptsExt{
			CreateOptsBuilder:   opts,
			PortSecurityEnabled: &val,
		}
	}

	port, err := ports.Create(cmd.Context(), c, createOpts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, port)
}

// runUpdate applies at least one update option and renders the updated port.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts := ports.UpdateOpts{}
	applied := false
	var securityEnabled *bool

	if appendAap, _ := cmd.Flags().GetBool("append-aap"); appendAap && !cmd.Flags().Changed("allowed-address-pair") {
		return fmt.Errorf("--append-aap requires --allowed-address-pair")
	}

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
	if cmd.Flags().Changed("admin-state-up") {
		val, _ := cmd.Flags().GetBool("admin-state-up")
		opts.AdminStateUp = &val
		applied = true
	}
	if cmd.Flags().Changed("device-id") {
		val := mustString(cmd, "device-id")
		opts.DeviceID = &val
		applied = true
	}
	if cmd.Flags().Changed("device-owner") {
		val := mustString(cmd, "device-owner")
		opts.DeviceOwner = &val
		applied = true
	}
	if cmd.Flags().Changed("allowed-address-pair") {
		pairs := addressPairs(cmd)
		if appendAap, _ := cmd.Flags().GetBool("append-aap"); appendAap {
			existing, err := fetchExistingAAPs(cmd, c, args[0])
			if err != nil {
				return err
			}
			pairs = append(existing, pairs...)
		}
		opts.AllowedAddressPairs = &pairs
		applied = true
	}
	if cmd.Flags().Changed("port-security-enabled") {
		val, _ := cmd.Flags().GetBool("port-security-enabled")
		securityEnabled = &val
		applied = true
	}
	if !applied {
		return fmt.Errorf("at least one update option is required: --name, --description, --admin-state-up, --device-id, --device-owner, --port-security-enabled or --allowed-address-pair")
	}

	var updateOpts ports.UpdateOptsBuilder = opts
	if securityEnabled != nil {
		updateOpts = portsecurity.PortUpdateOptsExt{
			UpdateOptsBuilder:   opts,
			PortSecurityEnabled: securityEnabled,
		}
	}

	port, err := ports.Update(cmd.Context(), c, args[0], updateOpts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, port)
}

// runDelete removes a port by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := ports.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "port %s deleted\n", args[0])
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

// addressPairs parses the repeatable --allowed-address-pair flag into
// ports.AddressPair values. Each entry is either "IP" or "IP,MAC".
func addressPairs(cmd *cobra.Command) []ports.AddressPair {
	raw, _ := cmd.Flags().GetStringSlice("allowed-address-pair")
	pairs := make([]ports.AddressPair, 0, len(raw))
	for _, r := range raw {
		parts := strings.Split(r, ",")
		pair := ports.AddressPair{IPAddress: parts[0]}
		if len(parts) > 1 {
			pair.MACAddress = parts[1]
		}
		pairs = append(pairs, pair)
	}
	return pairs
}

// fetchExistingAAPs returns the port's current allowed address pairs so that
// --append-aap can extend them instead of replacing the whole list.
func fetchExistingAAPs(cmd *cobra.Command, c *gophercloud.ServiceClient, id string) ([]ports.AddressPair, error) {
	p, err := ports.Get(cmd.Context(), c, id).Extract()
	if err != nil {
		return nil, err
	}
	return p.AllowedAddressPairs, nil
}

// fixedIPs parses the repeatable --fixed-ip flag into ports.IP values. Each
// entry is a comma-separated list of "subnet=<id>" and "ip-address=<ip>"
// keys; at least one of them must be present.
func fixedIPs(cmd *cobra.Command) ([]ports.IP, error) {
	raw, _ := cmd.Flags().GetStringArray("fixed-ip")
	ips := make([]ports.IP, 0, len(raw))
	for _, r := range raw {
		var ip ports.IP
		set := false
		for _, part := range strings.Split(r, ",") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("invalid --fixed-ip entry %q: expected subnet=<subnet-id>,ip-address=<ip-address>", r)
			}
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			switch key {
			case "subnet":
				ip.SubnetID = value
				set = true
			case "ip-address":
				ip.IPAddress = value
				set = true
			default:
				return nil, fmt.Errorf("invalid --fixed-ip key %q in %q: expected subnet or ip-address", key, r)
			}
		}
		if !set {
			return nil, fmt.Errorf("invalid --fixed-ip entry %q: at least one of subnet or ip-address is required", r)
		}
		ips = append(ips, ip)
	}
	return ips, nil
}
