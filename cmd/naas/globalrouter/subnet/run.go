/*
Copyright © 2026 Kot1gruN
*/
package subnet

import (
	"fmt"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/output"
)

// runList lists subnets, applying the --name filter when set, and renders the
// result through the --output flag. The Global Router subnet list API is
// account/project-wide rather than regional.
func runList(cmd *cobra.Command, c *v1.ServiceClient) error {
	q := &v1.SubnetsQueryParams{}
	if name, _ := cmd.Flags().GetString("name"); name != "" {
		q.Filters.Name = name
	}

	list, _, err := c.Subnets(cmd.Context(), q)
	if err != nil {
		return err
	}
	if len(*list) == 0 && output.IsTable(cmd) {
		return output.PrintListEmpty(cmd, "")
	}
	return render(cmd, *list)
}

// runShow fetches a single subnet by its ID and renders its full attributes
// through the --output flag.
func runShow(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	subnet, _, err := c.Subnet(cmd.Context(), id)
	if err != nil {
		return err
	}
	return renderShow(cmd, subnet)
}

// runCreate builds the create options from the flags and renders the created
// subnet. network-id and cidr are required. When --os-subnet-id is set the VPC
// flow is used (DedicatedSubnetCreate otherwise).
func runCreate(cmd *cobra.Command, c *v1.ServiceClient) error {
	networkID, _ := cmd.Flags().GetString("network-id")
	cidr, _ := cmd.Flags().GetString("cidr")
	if networkID == "" {
		return fmt.Errorf("network-id is required: pass --network-id")
	}
	if cidr == "" {
		return fmt.Errorf("cidr is required: pass --cidr")
	}

	gateway, _ := cmd.Flags().GetString("gateway")
	osSubnetID, _ := cmd.Flags().GetString("os-subnet-id")
	name, _ := cmd.Flags().GetString("name")
	projectID, _ := cmd.Flags().GetString("project-id")
	serviceAddresses, _ := cmd.Flags().GetStringSlice("service-addresses")
	tags, _ := cmd.Flags().GetStringSlice("tags")

	var subnet *v1.Subnet
	if osSubnetID != "" {
		req := &v1.VPCSubnetCreateRequest{
			NetworkID:        networkID,
			Gateway:          gateway,
			Cidr:             cidr,
			OsSubnetID:       osSubnetID,
			Name:             name,
			ProjectID:        projectID,
			ServiceAddresses: serviceAddresses,
			Tags:             tags,
		}
		s, _, err := c.VPCSubnetCreate(cmd.Context(), req)
		if err != nil {
			return err
		}
		subnet = s
	} else {
		req := &v1.DedicatedSubnetCreateRequest{
			NetworkID:        networkID,
			Gateway:          gateway,
			Cidr:             cidr,
			Name:             name,
			ServiceAddresses: serviceAddresses,
			Tags:             tags,
		}
		s, _, err := c.DedicatedSubnetCreate(cmd.Context(), req)
		if err != nil {
			return err
		}
		subnet = s
	}
	return render(cmd, *subnet)
}

// runUpdate applies at least one update option (name/tags) and renders the
// updated subnet.
func runUpdate(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	req := &v1.SubnetUpdateRequest{}
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

	subnet, _, err := c.SubnetUpdate(cmd.Context(), id, req)
	if err != nil {
		return err
	}
	return render(cmd, *subnet)
}

// runDelete disconnects and deletes a subnet by its ID, printing a confirmation
// line. The Global Router API deletes a subnet through the disconnect endpoint.
func runDelete(cmd *cobra.Command, c *v1.ServiceClient, id string) error {
	if _, err := c.SubnetDisconnect(cmd.Context(), id); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "subnet %s disconnected\n", id)
	return nil
}
