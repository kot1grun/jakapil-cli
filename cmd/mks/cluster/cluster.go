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
	"jakapil-cli/cmd/mks/output"
)

// NewClusterCommand returns the "cluster" subcommand.
// Without arguments it lists all clusters; with "--id" it returns
// a single cluster referenced by its identifier.
// The "--output" flag controls the rendering format: json, yaml or
// the default plain-text representation.
func NewClusterCommand() *cobra.Command {
	var (
		clusterID string
		outFormat string
	)

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "List clusters or get a single cluster by ID",
		Long: `List all Kubernetes clusters in the current project, or get
details of a single cluster by its identifier.

Examples:
  jakapil-cli mks cluster
  jakapil-cli mks cluster --id <cluster-id>
  jakapil-cli mks cluster --id <cluster-id> --output json
  jakapil-cli mks cluster --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			if clusterID != "" {
				cl, _, err := mkscluster.Get(ctx, mksClient, clusterID)
				if err != nil {
					return err
				}
				return renderCluster(outFormat, cl)
			}

			clusters, _, err := mkscluster.List(ctx, mksClient)
			if err != nil {
				return err
			}
			return renderClusters(outFormat, clusters)
		},
	}

	cmd.Flags().StringVarP(&clusterID, "id", "i", "", "cluster ID to get details for")
	cmd.Flags().StringVarP(&outFormat, "output", "o", output.YAML, "output format: json or yaml")

	return cmd
}

// renderCluster outputs a single cluster using the requested format.
func renderCluster(format string, cl *mkscluster.GetView) error {
	printed, err := output.Render(format, cl)
	if err != nil {
		return err
	}
	if !printed {
		fmt.Printf("%+v\n", cl)
	}

	return nil
}

// renderClusters outputs a list of clusters using the requested format.
func renderClusters(format string, clusters []*mkscluster.ListView) error {
	printed, err := output.Render(format, clusters)
	if err != nil {
		return err
	}
	if !printed {
		for _, cl := range clusters {
			fmt.Printf("%+v\n", cl)
		}
	}

	return nil
}
