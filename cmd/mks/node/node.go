/*
Copyright © 2026 Kot1gruN
*/
package node

import (
	"context"
	"fmt"

	"github.com/selectel/mks-go/pkg/v1"
	mksnode "github.com/selectel/mks-go/pkg/v1/node"
	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
)

// NewNodeCommand returns the "node" subcommand.
// Without arguments it shows the command help; "list" lists nodes of a
// cluster; with "--cluster-id", "--nodegroup-id" and "--id" it returns a
// single node.
func NewNodeCommand() *cobra.Command {
	var (
		clusterID   string
		nodegroupID string
		nodeID      string
	)

	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage cluster nodes (list, get)",
		Long: `Manage Kubernetes nodes of a cluster nodegroup.

Subcommands:
  list  List nodes of a cluster (optionally of a single nodegroup)

Flags:
  -i, --id  Get details of a single node

Examples:
  jakapil-cli mks node list --cluster-id <cluster-id>
  jakapil-cli mks node --cluster-id <cluster-id> --nodegroup-id <nodegroup-id> --id <node-id>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" || nodegroupID == "" || nodeID == "" {
				return cmd.Help()
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			n, _, err := mksnode.Get(ctx, mksClient, clusterID, nodegroupID, nodeID)
			if err != nil {
				return err
			}
			return render(cmd, n)
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVar(&nodegroupID, "nodegroup-id", "", "nodegroup ID")
	cmd.Flags().StringVarP(&nodeID, "id", "i", "", "node ID")
	cmd.AddCommand(newListCommand())

	return cmd
}

// newListCommand returns the "node list" subcommand. There is no dedicated
// node-list API, so nodes are aggregated from the nodegroups of the cluster.
func newListCommand() *cobra.Command {
	var (
		clusterID   string
		nodegroupID string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List nodes of a cluster",
		Long: `List nodes of a cluster nodegroup. Nodes are aggregated from the
nodegroups of the cluster. Use "--nodegroup-id" to list only the nodes of a
single nodegroup.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}
			if allRegions, _ := cmd.Flags().GetBool(client.FlagAllRegions); allRegions {
				return runListAllRegions(cmd, clusterID, nodegroupID)
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			nodes, err := listNodesInRegion(ctx, mksClient, clusterID, nodegroupID)
			if err != nil {
				return err
			}
			if len(nodes) == 0 && output.IsTable(cmd) {
				return output.PrintListEmpty(cmd, client.EffectiveRegion(cmd, mksClient))
			}
			return render(cmd, nodes)
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVar(&nodegroupID, "nodegroup-id", "", "nodegroup ID")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list nodes in every region the service advertises instead of --region")

	return cmd
}

// listNodesInRegion aggregates the nodes of a cluster, optionally restricted to
// a single nodegroup, within one region. There is no dedicated node-list API,
// so nodes are aggregated from the nodegroups of the cluster.
func listNodesInRegion(ctx context.Context, c *v1.ServiceClient, clusterID, nodegroupID string) ([]*mksnode.View, error) {
	if nodegroupID != "" {
		ng, _, err := mksnodegroup.Get(ctx, c, clusterID, nodegroupID)
		if err != nil {
			return nil, err
		}
		return ng.Nodes, nil
	}

	groups, _, err := mksnodegroup.List(ctx, c, clusterID)
	if err != nil {
		return nil, err
	}
	nodes := make([]*mksnode.View, 0)
	for _, ng := range groups {
		nodes = append(nodes, ng.Nodes...)
	}
	return nodes, nil
}

// runListAllRegions lists nodes of a cluster in every region the service
// advertises and renders the aggregated result with a "Region" column (table)
// or grouped by region (json/yaml), backed by the --all-regions flag. Regions
// are queried in parallel with parallel.Regions, since the underlying calls are
// read-only GETs. A region whose endpoint is unavailable is skipped with a
// warning on stderr; the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command, clusterID, nodegroupID string) error {
	regions, clients, err := client.NewClients(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()

	results := parallel.Regions(ctx, regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		nodes, err := listNodesInRegion(ctx, clients[region], clusterID, nodegroupID)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(nodes))
		for _, n := range nodes {
			rows = append(rows, nodeRow(n))
		}
		return rows, nodes, nil
	})

	return output.RenderRegionsBestEffort(cmd, nodeHeader, results)
}
