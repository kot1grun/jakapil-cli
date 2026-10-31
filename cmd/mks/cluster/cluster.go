/*
Copyright © 2026 Kot1gruN
*/
package cluster

import (
	"context"

	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
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
	cmd := &cobra.Command{
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
	cmd.Flags().Bool(client.FlagAllRegions, false, "list clusters in every region the service advertises instead of --region")
	return cmd
}

// runListAllRegions lists clusters in every region the service advertises and
// renders the aggregated result with a query "Region" column (table) or grouped
// by region (json/yaml), backed by the --all-regions flag. Regions are queried
// in parallel with parallel.Regions, since the underlying calls are read-only
// GETs. A region whose endpoint is unavailable is skipped with a warning on
// stderr; the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command) error {
	regions, clients, err := client.NewClients(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()

	results := parallel.Regions(ctx, regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		clusters, _, err := mkscluster.List(ctx, clients[region])
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(clusters))
		for _, cl := range clusters {
			rows = append(rows, clusterListViewRow(cl))
		}
		return rows, clusters, nil
	})

	return output.RenderRegionsBestEffort(cmd, clusterHeader, results)
}
