/*
Copyright © 2026 Kot1gruN
*/
package cluster

import (
	"context"
	"fmt"

	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
	"jakapil-cli/internal/output"
)

// NewClusterCommand returns the "cluster" subcommand.
// Without arguments it shows the command help; with "list" it lists all
// clusters; with "--id" it returns a single cluster by its identifier.
// The persistent "--output" flag controls the rendering format.
func NewClusterCommand() *cobra.Command {
	var clusterID string

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage MKS clusters (list, get)",
		Long: `Manage Kubernetes clusters of the Selectel Managed Kubernetes Service.

Subcommands:
  list            List all clusters in the current project
  create          Create a new cluster
  delete          Delete a cluster by its identifier
  get-kubeconfig  Get the kubeconfig of a cluster by its identifier
  rotate-certs    Rotate the certificates of a cluster by its identifier
  task            Manage cluster tasks (list, show)

Flags:
  -i, --id  Get details of a single cluster by its identifier

Examples:
  jakapil-cli mks cluster list
  jakapil-cli mks cluster --id <cluster-id>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return cmd.Help()
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			cl, _, err := mkscluster.Get(ctx, mksClient, clusterID)
			if err != nil {
				return err
			}
			return render(cmd, cl)
		},
	}

	cmd.Flags().StringVarP(&clusterID, "id", "i", "", "cluster ID to get details for")
	cmd.AddCommand(
		newListCommand(),
		newCreateCommand(),
		newDeleteCommand(),
		newGetKubeconfigCommand(),
		newRotateCertsCommand(),
		newTaskCommand(),
	)

	return cmd
}

// newListCommand returns the "cluster list" subcommand.
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all clusters",
		Long:  "List all Kubernetes clusters in the current project.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRegions, _ := cmd.Flags().GetBool(client.FlagAllRegions); allRegions {
				return runListAllRegions(cmd)
			}
			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			clusters, _, err := mkscluster.List(ctx, mksClient)
			if err != nil {
				return err
			}
			return render(cmd, clusters)
		},
	}
}

// runListAllRegions lists clusters in every region the service advertises and
// renders the aggregated result with a query "Region" column (table) or grouped
// by region (json/yaml), backed by the --all-regions flag.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewClients(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()

	regionRows := make([]output.RegionRows, 0, len(regions))
	grouped := make([]output.RegionValue, 0, len(regions))
	for _, region := range regions {
		clusters, _, err := mkscluster.List(ctx, clients[region])
		if err != nil {
			return fmt.Errorf("region %s: %w", region, err)
		}
		rows := make([][]string, 0, len(clusters))
		for _, cl := range clusters {
			rows = append(rows, clusterListViewRow(cl))
		}
		regionRows = append(regionRows, output.RegionRows{Region: region, Rows: rows})
		grouped = append(grouped, output.RegionValue{Region: region, Value: clusters})
	}
	return output.RenderRegions(cmd, clusterHeader, regionRows, grouped)
}
