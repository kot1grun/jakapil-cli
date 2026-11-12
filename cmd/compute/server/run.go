/*
Copyright © 2026 Kot1gruN
*/
package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists servers and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchServers(cmd.Context(), c, serverListOpts(cmd))
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runShow fetches a single server by its ID and renders it through the --output
// flag.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	server, err := servers.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	return renderShow(cmd, server)
}

// runListAllRegions lists servers in every region the service advertises and
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
	opts := serverListOpts(cmd)

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchServers(ctx, clients[region], opts)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, s := range list {
			rows = append(rows, serverRow(s))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, serverHeader, results)
}

// serverListOpts builds the servers list options from the list flags.
func serverListOpts(cmd *cobra.Command) servers.ListOpts {
	opts := servers.ListOpts{}
	if v, _ := cmd.Flags().GetString("changes-since"); v != "" {
		opts.ChangesSince = v
	}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		opts.Name = v
	}
	if v, _ := cmd.Flags().GetString("status"); v != "" {
		opts.Status = v
	}
	if v, _ := cmd.Flags().GetString("image"); v != "" {
		opts.Image = v
	}
	if v, _ := cmd.Flags().GetString("flavor"); v != "" {
		opts.Flavor = v
	}
	if v, _ := cmd.Flags().GetString("ip"); v != "" {
		opts.IP = v
	}
	if v, _ := cmd.Flags().GetString("ip6"); v != "" {
		opts.IP6 = v
	}
	if v, _ := cmd.Flags().GetString("host"); v != "" {
		opts.Host = v
	}
	if v, _ := cmd.Flags().GetInt("limit"); v > 0 {
		opts.Limit = v
	}
	if v, _ := cmd.Flags().GetString("marker"); v != "" {
		opts.Marker = v
	}
	if v, _ := cmd.Flags().GetString("tags"); v != "" {
		opts.Tags = v
	}
	if v, _ := cmd.Flags().GetString("tags-any"); v != "" {
		opts.TagsAny = v
	}
	if v, _ := cmd.Flags().GetString("not-tags"); v != "" {
		opts.NotTags = v
	}
	if v, _ := cmd.Flags().GetString("not-tags-any"); v != "" {
		opts.NotTagsAny = v
	}
	if v, _ := cmd.Flags().GetString("availability-zone"); v != "" {
		opts.AvailabilityZone = v
	}
	return opts
}

// fetchServers lists servers through the paginator and returns them flat.
func fetchServers(ctx context.Context, c *gophercloud.ServiceClient, opts servers.ListOpts) ([]servers.Server, error) {
	var list []servers.Server
	pager := servers.List(c, opts)
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageServers, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageServers...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// runCreate creates a server and renders the (accepted) create response.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	opts, err := serverCreateOpts(cmd)
	if err != nil {
		return err
	}

	server, err := servers.Create(cmd.Context(), c, opts, nil).Extract()
	if err != nil {
		return err
	}
	return renderShow(cmd, server)
}

// serverCreateOpts builds the servers.CreateOpts from the create flags,
// returning an error when required flags are missing or mutually exclusive
// network options are combined. When --key-name is set the opts are wrapped in
// a keypairs.CreateOptsExt so the key_name field lands in the request body.
func serverCreateOpts(cmd *cobra.Command) (servers.CreateOptsBuilder, error) {
	opts := servers.CreateOpts{
		Name:             mustString(cmd, "name"),
		ImageRef:         mustString(cmd, "image-ref"),
		FlavorRef:        mustString(cmd, "flavor-ref"),
		AvailabilityZone: mustString(cmd, "availability-zone"),
	}
	if opts.Name == "" {
		return opts, errors.New("--name is required")
	}
	if opts.FlavorRef == "" {
		return opts, errors.New("--flavor-ref is required")
	}

	if v, _ := cmd.Flags().GetStringArray("security-group"); len(v) > 0 {
		opts.SecurityGroups = v
	}
	if v, _ := cmd.Flags().GetStringToString("metadata"); len(v) > 0 {
		opts.Metadata = v
	}
	if v, _ := cmd.Flags().GetString("admin-pass"); v != "" {
		opts.AdminPass = v
	}
	if v, _ := cmd.Flags().GetString("user-data-file"); v != "" {
		data, err := os.ReadFile(v)
		if err != nil {
			return opts, fmt.Errorf("read --user-data-file %q: %w", v, err)
		}
		opts.UserData = data
	}
	if apply, drive, err := configDriveFlag(cmd); err != nil {
		return opts, err
	} else if apply {
		opts.ConfigDrive = &drive
	}
	if v, _ := cmd.Flags().GetString("disk-config"); v != "" {
		opts.DiskConfig = servers.DiskConfig(strings.ToUpper(v))
	}
	if v, _ := cmd.Flags().GetInt("min-count"); v > 0 {
		opts.Min = v
	}
	if v, _ := cmd.Flags().GetInt("max-count"); v > 0 {
		opts.Max = v
	}
	if v, _ := cmd.Flags().GetStringArray("tag"); len(v) > 0 {
		opts.Tags = v
	}

	networks, err := networkOpts(cmd)
	if err != nil {
		return opts, err
	}
	opts.Networks = networks

	if keyName, _ := cmd.Flags().GetString("key-name"); keyName != "" {
		return keypairs.CreateOptsExt{CreateOptsBuilder: opts, KeyName: keyName}, nil
	}

	return opts, nil
}

// configDriveFlag resolves the --config-drive / --config-drive-off pair into
// (apply, enabled).
func configDriveFlag(cmd *cobra.Command) (bool, bool, error) {
	on := cmd.Flags().Changed("config-drive")
	off := cmd.Flags().Changed("config-drive-off")
	if on && off {
		return false, false, errors.New("--config-drive and --config-drive-off are mutually exclusive")
	}
	if on {
		v, _ := cmd.Flags().GetBool("config-drive")
		return true, v, nil
	}
	if off {
		return true, false, nil
	}
	return false, false, nil
}

// networkOpts resolves the --net/--port lists or the --network auto/none string
// into the value accepted by servers.CreateOpts.Networks.
func networkOpts(cmd *cobra.Command) (any, error) {
	auto, _ := cmd.Flags().GetString("network")
	nets, _ := cmd.Flags().GetStringArray("net")
	ports, _ := cmd.Flags().GetStringArray("port")

	if auto != "" {
		if len(nets) > 0 || len(ports) > 0 {
			return nil, errors.New("--network is mutually exclusive with --net/--port")
		}
		if auto != "auto" && auto != "none" {
			return nil, fmt.Errorf("--network must be \"auto\" or \"none\", got %q", auto)
		}
		return auto, nil
	}

	if len(nets) == 0 && len(ports) == 0 {
		// The default of Nova is to attach the server to all isolated networks
		// of the tenant, so leaving the field empty is valid.
		return nil, nil
	}

	networks := make([]servers.Network, 0, len(nets)+len(ports))
	for _, n := range nets {
		networks = append(networks, servers.Network{UUID: n})
	}
	for _, p := range ports {
		networks = append(networks, servers.Network{Port: p})
	}
	return networks, nil
}

// runUpdate applies at least one update option and renders the updated server.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts := servers.UpdateOpts{}
	changed := false
	if cmd.Flags().Changed("name") {
		opts.Name = mustString(cmd, "name")
		changed = true
	}
	if cmd.Flags().Changed("access-ipv4") {
		opts.AccessIPv4 = mustString(cmd, "access-ipv4")
		changed = true
	}
	if cmd.Flags().Changed("access-ipv6") {
		opts.AccessIPv6 = mustString(cmd, "access-ipv6")
		changed = true
	}
	if v := mustString(cmd, "hostname"); cmd.Flags().Changed("hostname") {
		opts.Hostname = &v
		changed = true
	}
	if !changed {
		return fmt.Errorf("at least one update option is required: --name, --access-ipv4, --access-ipv6 or --hostname")
	}

	server, err := servers.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return renderShow(cmd, server)
}

// runDelete removes a server by its ID (soft by default, or via os-forceDelete
// with --force) and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	if force {
		if err := servers.ForceDelete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
			return err
		}
	} else {
		if err := servers.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
			return err
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "server %s deleted\n", args[0])
	return nil
}

// runPause suspends a virtual machine's vCPUs (os-pause) and prints a
// confirmation line.
func runPause(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := servers.Pause(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "server %s paused\n", args[0])
	return nil
}

// runUnpause resumes a paused virtual machine (os-unpause) and prints a
// confirmation line.
func runUnpause(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := servers.Unpause(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "server %s unpaused\n", args[0])
	return nil
}

// runReboot reboots a virtual machine (os-reboot). A soft reboot is used unless
// the --hard flag is set.
func runReboot(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	method := servers.SoftReboot
	if hard, _ := cmd.Flags().GetBool("hard"); hard {
		method = servers.HardReboot
	}
	opts := servers.RebootOpts{Type: method}
	if err := servers.Reboot(cmd.Context(), c, args[0], opts).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "server %s rebooted (%s)\n", args[0], method)
	return nil
}

// runStop cleanly shuts down a virtual machine (os-stop) and prints a
// confirmation line.
func runStop(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := servers.Stop(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "server %s stopped\n", args[0])
	return nil
}

// runStart powers on a stopped virtual machine (os-start) and prints a
// confirmation line.
func runStart(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := servers.Start(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "server %s started\n", args[0])
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
