/*
Copyright © 2026 Kot1gruN
*/
package globalrouter

import (
	"fmt"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// runList lists routers, applying the --name filter when set, and renders the
// result through the --output flag. The Global Router API is a NaaS product, so
// the listing is account/project-wide rather than regional.
func runList(cmd *cobra.Command, c *v1.ServiceClient) error {
	q := &v1.RoutersQueryParams{}
	if name, _ := cmd.Flags().GetString("name"); name != "" {
		q.Filters.Name = name
	}

	list, _, err := c.Routers(cmd.Context(), q)
	if err != nil {
		return err
	}
	if len(*list) == 0 && output.IsTable(cmd) {
		return output.PrintListEmpty(cmd, "")
	}
	return render(cmd, *list)
}

// runCreate builds the create options from the flags and renders the created
// router. The name is required.
func runCreate(cmd *cobra.Command, c *v1.ServiceClient) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		return fmt.Errorf("router name is required: pass --name")
	}
	tags, _ := cmd.Flags().GetStringSlice("tags")

	req := &v1.RouterCreateRequest{Name: name, Tags: tags}

	router, _, err := c.RouterCreate(cmd.Context(), req)
	if err != nil {
		return err
	}
	return render(cmd, *router)
}

// runUpdate applies at least one update option (name/tags) and renders the
// updated router.
func runUpdate(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	req := &v1.RouterUpdateRequest{}
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

	router, _, err := c.RouterUpdate(cmd.Context(), id, req)
	if err != nil {
		return err
	}
	return render(cmd, *router)
}

// runDelete removes a router by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	if _, err := c.RouterDelete(cmd.Context(), id); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "router %s deleted\n", id)
	return nil
}
