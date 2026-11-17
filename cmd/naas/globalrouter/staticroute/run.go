/*
Copyright © 2026 Kot1gruN
*/
package staticroute

import (
	"fmt"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// runList lists static routes, applying the --name (API) and --router-id
// (client-side) filters when set, and renders the result through the --output
// flag. The Global Router list API has no router filter, so filtering by router
// is done locally over the returned set.
func runList(cmd *cobra.Command, c *v1.ServiceClient) error {
	q := &v1.StaticRoutesQueryParams{}
	if name, _ := cmd.Flags().GetString("name"); name != "" {
		q.Filters.Name = name
	}
	routerID, _ := cmd.Flags().GetString("router-id")

	list, _, err := c.StaticRoutes(cmd.Context(), q)
	if err != nil {
		return err
	}

	routes := *list
	if routerID != "" {
		filtered := routes[:0]
		for _, r := range routes {
			if r.RouterID == routerID {
				filtered = append(filtered, r)
			}
		}
		routes = filtered
	}

	if len(routes) == 0 && output.IsTable(cmd) {
		return output.PrintListEmpty(cmd, "")
	}
	return render(cmd, routes)
}

// runShow fetches a single static route by its ID and renders its full
// attributes through the --output flag.
func runShow(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	route, _, err := c.StaticRoute(cmd.Context(), id)
	if err != nil {
		return err
	}
	return renderShow(cmd, route)
}

// runCreate builds the create options from the flags and renders the created
// static route. router-id, next-hop and cidr are required.
func runCreate(cmd *cobra.Command, c *v1.ServiceClient) error {
	routerID, _ := cmd.Flags().GetString("router-id")
	nextHop, _ := cmd.Flags().GetString("next-hop")
	cidr, _ := cmd.Flags().GetString("cidr")
	if routerID == "" {
		return fmt.Errorf("router-id is required: pass --router-id")
	}
	if nextHop == "" {
		return fmt.Errorf("next-hop is required: pass --next-hop")
	}
	if cidr == "" {
		return fmt.Errorf("cidr is required: pass --cidr")
	}

	req := &v1.StaticRouteCreateRequest{
		RouterID: routerID,
		NextHop:  nextHop,
		Cidr:     cidr,
	}
	if name, _ := cmd.Flags().GetString("name"); name != "" {
		req.Name = name
	}
	if projectID, _ := cmd.Flags().GetString("project-id"); projectID != "" {
		req.ProjectID = projectID
	}
	if tags, _ := cmd.Flags().GetStringSlice("tags"); len(tags) > 0 {
		req.Tags = tags
	}

	route, _, err := c.StaticRouteCreate(cmd.Context(), req)
	if err != nil {
		return err
	}
	return render(cmd, *route)
}

// runUpdate applies at least one update option (name/tags) and renders the
// updated static route.
func runUpdate(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	req := &v1.StaticRouteUpdateRequest{}
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

	route, _, err := c.StaticRouteUpdate(cmd.Context(), id, req)
	if err != nil {
		return err
	}
	return render(cmd, *route)
}

// runDelete removes a static route by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	if _, err := c.StaticRouteDelete(cmd.Context(), id); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "static route %s deleted\n", id)
	return nil
}
