/*
Copyright © 2026 Kot1gruN
*/
package port

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// runList lists ports, applying the --name, --network-id, --status,
// --mac-address, --device-id and --limit filters when set, and renders the
// result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
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

	var list []ports.Port
	pager := ports.List(c, opts)
	if err := pager.EachPage(cmd.Context(), func(_ context.Context, page pagination.Page) (bool, error) {
		pagePorts, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		list = append(list, pagePorts...)
		return true, nil
	}); err != nil {
		return err
	}

	return output.Print(cmd, list)
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

	port, err := ports.Create(cmd.Context(), c, opts).Extract()
	if err != nil {
		return err
	}
	return output.Print(cmd, port)
}

// runUpdate applies at least one update option and renders the updated port.
func runUpdate(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	opts := ports.UpdateOpts{}
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
	if !applied {
		return fmt.Errorf("at least one update option is required: --name, --description, --admin-state-up, --device-id or --device-owner")
	}

	port, err := ports.Update(cmd.Context(), c, args[0], opts).Extract()
	if err != nil {
		return err
	}
	return output.Print(cmd, port)
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
