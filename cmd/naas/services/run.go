/*
Copyright © 2026 Kot1gruN
*/
package services

import (
	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// runList lists services, applying the --name filter when set, and renders the
// result through the --output flag. The Global Router API is a NaaS product, so
// the listing is account/project-wide rather than regional.
func runList(cmd *cobra.Command, c *v1.ServiceClient) error {
	q := &v1.ServicesQueryParams{}
	if name, _ := cmd.Flags().GetString("name"); name != "" {
		q.Filters.Name = name
	}

	list, _, err := c.ListServices(cmd.Context(), q)
	if err != nil {
		return err
	}
	if len(*list) == 0 && output.IsTable(cmd) {
		return output.PrintListEmpty(cmd, "")
	}
	return render(cmd, *list)
}
