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

	"jakapil-cli/internal/output"
)

// runList lists networks, applying the --name, --status, --id and --limit
// filters when set, and renders the result through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
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

	var list []networks.Network
	pager := networks.List(c, opts)
	if err := pager.EachPage(cmd.Context(), func(_ context.Context, page pagination.Page) (bool, error) {
		pageNetworks, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageNetworks...)
		return true, nil
	}); err != nil {
		return err
	}

	return output.Print(cmd, list)
}

// runShow renders a single network selected by its ID.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	network, err := networks.Get(cmd.Context(), c, args[0]).Extract()
	if err != nil {
		return err
	}
	return output.Print(cmd, network)
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
	return output.Print(cmd, network)
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
	return output.Print(cmd, network)
}

// runDelete removes a network by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := networks.Delete(cmd.Context(), c, args[0]).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "network %s deleted\n", args[0])
	return nil
}
