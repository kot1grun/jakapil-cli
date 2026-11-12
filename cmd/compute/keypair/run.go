/*
Copyright © 2026 Kot1gruN
*/
package keypair

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/compute/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// runList lists the project keypairs and renders them through the --output flag.
func runList(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	list, err := fetchKeypairs(cmd.Context(), c)
	if err != nil {
		return err
	}
	return render(cmd, list)
}

// runListAllRegions lists keypairs in every region the service advertises and
// renders the aggregated result with a Region column (table) or grouped by
// region (json/yaml), backed by the --all-regions flag. Regions are queried in
// parallel with parallel.Regions, since the underlying calls are read-only GETs.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewComputeServiceClients(cmd)
	if err != nil {
		return err
	}

	results := parallel.Regions(cmd.Context(), regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		list, err := fetchKeypairs(ctx, clients[region])
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(list))
		for _, k := range list {
			rows = append(rows, keypairRow(k))
		}
		return rows, list, nil
	})

	return output.RenderRegionsBestEffort(cmd, keypairHeader, results)
}

// fetchKeypairs lists keypairs through the paginator and returns them flat.
func fetchKeypairs(ctx context.Context, c *gophercloud.ServiceClient) ([]keypairs.KeyPair, error) {
	var list []keypairs.KeyPair
	pager := keypairs.List(c, keypairs.ListOpts{})
	if err := pager.EachPage(ctx, func(_ context.Context, page pagination.Page) (bool, error) {
		pageKeys, err := keypairs.ExtractKeyPairs(page)
		if err != nil {
			return false, err
		}
		list = append(list, pageKeys...)
		return true, nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// runShow fetches and renders a single keypair by name.
func runShow(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	kp, err := keypairs.Get(cmd.Context(), c, args[0], keypairs.GetOpts{}).Extract()
	if err != nil {
		return err
	}
	return render(cmd, kp)
}

// runCreate creates (or imports) a keypair. The --name flag is required; when
// --public-key is given the key is imported instead of generated.
func runCreate(cmd *cobra.Command, c *gophercloud.ServiceClient) error {
	opts := keypairs.CreateOpts{
		Name: mustString(cmd, "name"),
	}
	if opts.Name == "" {
		return fmt.Errorf("--name is required")
	}
	if v := mustString(cmd, "public-key"); v != "" {
		opts.PublicKey = v
	}
	if v := mustString(cmd, "type"); v != "" {
		opts.Type = v
	}

	kp, err := keypairs.Create(cmd.Context(), c, opts).Extract()
	if err != nil {
		return err
	}
	return render(cmd, kp)
}

// runDelete removes a keypair by name and prints a confirmation line.
func runDelete(cmd *cobra.Command, c *gophercloud.ServiceClient, args []string) error {
	if err := keypairs.Delete(cmd.Context(), c, args[0], keypairs.DeleteOpts{}).ExtractErr(); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "keypair %s deleted\n", args[0])
	return nil
}

// mustString reads a string-typed flag value.
func mustString(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}
