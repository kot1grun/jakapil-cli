/*
Copyright © 2026 Kot1gruN
*/
package network

import (
	"fmt"
	"strings"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// runList lists networks, applying the --name filter when set, and renders the
// result through the --output flag.
func runList(cmd *cobra.Command, c *v1.ServiceClient) error {
	q := &v1.NetworksQueryParams{}
	if name, _ := cmd.Flags().GetString("name"); name != "" {
		q.Filters.Name = name
	}

	list, _, err := c.Networks(cmd.Context(), q)
	if err != nil {
		return err
	}
	if len(*list) == 0 && output.IsTable(cmd) {
		return output.PrintListEmpty(cmd, "")
	}
	return render(cmd, *list)
}

// runShow fetches a single network by its ID and renders its full attributes
// through the --output flag.
func runShow(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	net, _, err := c.Network(cmd.Context(), id)
	if err != nil {
		return err
	}
	return renderShow(cmd, net)
}

// runCreate builds the create options from the flags and renders the created
// network. It dispatches to the VPC or dedicated create request depending on
// the --type value.
func runCreate(cmd *cobra.Command, c *v1.ServiceClient) error {
	routerID, _ := cmd.Flags().GetString("router-id")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	if routerID == "" {
		return fmt.Errorf("router-id is required: pass --router-id")
	}
	if zoneID == "" {
		return fmt.Errorf("zone-id is required: pass --zone-id")
	}

	name, _ := cmd.Flags().GetString("name")
	tags, _ := cmd.Flags().GetStringSlice("tags")

	typ, _ := cmd.Flags().GetString("type")
	var (
		net *v1.Network
		err error
	)
	switch strings.ToLower(typ) {
	case "", "vpc":
		req := &v1.VPCNetworkCreateRequest{
			RouterID: routerID,
			ZoneID:   zoneID,
			Name:     name,
			Tags:     tags,
		}
		if osNetworkID, _ := cmd.Flags().GetString("os-network-id"); osNetworkID != "" {
			req.OsNetworkID = osNetworkID
		} else {
			return fmt.Errorf("os-network-id is required for vpc networks: pass --os-network-id")
		}
		if projectID, _ := cmd.Flags().GetString("project-id"); projectID != "" {
			req.ProjectID = projectID
		}
		net, _, err = c.VPCNetworkCreate(cmd.Context(), req)
	case "dedicated":
		vlan, _ := cmd.Flags().GetInt("vlan")
		if vlan == 0 {
			return fmt.Errorf("vlan is required for dedicated networks: pass --vlan")
		}
		req := &v1.DedicatedNetworkCreateRequest{
			RouterID: routerID,
			ZoneID:   zoneID,
			Vlan:     vlan,
			Name:     name,
			Tags:     tags,
		}
		if innerVlan, _ := cmd.Flags().GetInt("inner-vlan"); innerVlan > 0 {
			req.InnerVlan = innerVlan
		}
		net, _, err = c.DedicatedNetworkCreate(cmd.Context(), req)
	default:
		return fmt.Errorf("unknown network type %q: must be vpc or dedicated", typ)
	}
	if err != nil {
		return err
	}
	return render(cmd, *net)
}

// runUpdate applies at least one update option (name/tags) and renders the
// updated network.
func runUpdate(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	req := &v1.NetworkUpdateRequest{}
	applied := false

	if cmd.Flags().Changed("name") {
		val, _ := cmd.Flags().GetString("name")
		req.Name = &val
		applied = true
	}
	if cmd.Flags().Changed("tags") {
		val, _ := cmd.Flags().GetStringSlice("tags")
		req.Tags = &val
		applied = true
	}
	if !applied {
		return fmt.Errorf("at least one update option is required: --name or --tags")
	}

	net, _, err := c.NetworkUpdate(cmd.Context(), id, req)
	if err != nil {
		return err
	}
	return render(cmd, *net)
}

// runDelete disconnects a network by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	if _, err := c.NetworkDisconnect(cmd.Context(), id); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "network %s disconnected\n", id)
	return nil
}
