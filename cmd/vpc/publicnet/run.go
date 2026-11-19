/*
Copyright © 2026 Kot1gruN
*/
package publicnet

import (
	"fmt"

	publicnetapi "github.com/selectel/public-net-api-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/publicnet/client"
	"jakapil-cli/internal/output"
)

// runList lists public ports, applying the --project-id, --network-id and
// --description filters when set, and renders the result through the --output
// flag.
func runList(cmd *cobra.Command, c *publicnetapi.PublicNetAPIClient) error {
	query, err := portsQuery(cmd)
	if err != nil {
		return err
	}

	list, err := c.ListPorts(cmd.Context(), query)
	if err != nil {
		return err
	}
	if len(list) == 0 && output.IsTable(cmd) {
		return output.PrintListEmpty(cmd, "")
	}
	return render(cmd, list)
}

// portsQuery builds the ports list query from the list flags.
func portsQuery(cmd *cobra.Command) (*publicnetapi.PortsQuery, error) {
	query := &publicnetapi.PortsQuery{}
	if v, _ := cmd.Flags().GetString("project-id"); v != "" {
		query.ProjectID = v
	}
	if v, _ := cmd.Flags().GetString("network-id"); v != "" {
		query.NetworkID = v
	}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		query.Description = v
	}
	return query, nil
}

// runListAllRegions lists public ports in every region the service advertises
// and renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewClients(cmd)
	if err != nil {
		return err
	}

	regionRows := make([]output.RegionRows, 0, len(regions))
	grouped := make([]output.RegionValue, 0, len(regions))

	for i, region := range regions {
		query, err := portsQuery(cmd)
		if err != nil {
			return err
		}
		list, err := clients[i].ListPorts(cmd.Context(), query)
		if err != nil {
			// A region whose endpoint is unavailable is skipped with a warning.
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: region %s: %v\n", region, err)
			continue
		}
		rows := make([][]string, 0, len(list))
		for _, p := range list {
			rows = append(rows, portRow(p))
		}
		regionRows = append(regionRows, output.RegionRows{Region: region, Rows: rows})
		grouped = append(grouped, output.RegionValue{Region: region, Value: list})
	}

	return output.RenderRegions(cmd, portHeader, regionRows, grouped)
}

// runShow renders a single public port selected by its ID.
func runShow(cmd *cobra.Command, c *publicnetapi.PublicNetAPIClient, id string) error {
	port, err := c.GetPort(cmd.Context(), id)
	if err != nil {
		return err
	}
	return renderShow(cmd, port)
}

// runCreate builds the create options from the flags and renders the created
// public port. The project ID comes from --project-id, or is inferred from the
// project scope of the token.
func runCreate(cmd *cobra.Command, c *publicnetapi.PublicNetAPIClient) error {
	projectID, _ := cmd.Flags().GetString("project-id")
	if projectID == "" {
		inferred, err := client.ResolveProjectID(cmd)
		if err != nil {
			return fmt.Errorf("inferring project ID from the token: %w", err)
		}
		if inferred == "" {
			return fmt.Errorf("project ID is required: pass --project-id or use a project-scoped token")
		}
		projectID = inferred
	}

	dto := &publicnetapi.PortCreateDTO{
		ProjectID: projectID,
	}
	if desc, _ := cmd.Flags().GetString("description"); desc != "" {
		dto.Description = &desc
	}
	if groups, _ := cmd.Flags().GetStringSlice("security-group-id"); groups != nil {
		dto.SecurityGroupIDs = groups
	}
	if adminUp, _ := cmd.Flags().GetBool("admin-state-up"); cmd.Flags().Changed("admin-state-up") {
		dto.AdminStateUp = &adminUp
	}

	port, err := c.CreatePort(cmd.Context(), dto)
	if err != nil {
		return err
	}
	return render(cmd, port)
}

// runUpdate applies at least one update option and renders the updated public
// port.
func runUpdate(cmd *cobra.Command, c *publicnetapi.PublicNetAPIClient, id string) error {
	dto := &publicnetapi.PortUpdateDTO{}
	applied := false

	if cmd.Flags().Changed("description") {
		val, _ := cmd.Flags().GetString("description")
		dto.Description = &val
		applied = true
	}
	if cmd.Flags().Changed("security-group-id") {
		val, _ := cmd.Flags().GetStringSlice("security-group-id")
		dto.SecurityGroupIDs = val
		applied = true
	}
	if cmd.Flags().Changed("admin-state-up") {
		val, _ := cmd.Flags().GetBool("admin-state-up")
		dto.AdminStateUp = &val
		applied = true
	}
	if !applied {
		return fmt.Errorf("at least one update option is required: --description, --security-group-id or --admin-state-up")
	}

	port, err := c.UpdatePort(cmd.Context(), id, dto)
	if err != nil {
		return err
	}
	return render(cmd, port)
}

// runDelete removes a public port by its ID and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *publicnetapi.PublicNetAPIClient, id string) error {
	query := &publicnetapi.PortDeleteQuery{}
	if force, _ := cmd.Flags().GetBool("force"); force {
		query.Force = true
	}
	if err := c.DeletePort(cmd.Context(), id, query); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "public port %s deleted\n", id)
	return nil
}
