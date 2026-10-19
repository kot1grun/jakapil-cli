/*
Copyright © 2026 Kot1gruN
*/
package node

import (
	"context"
	"fmt"

	mksnode "github.com/selectel/mks-go/pkg/v1/node"
	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
	"jakapil-cli/cmd/mks/output"
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
			return output.Print(cmd, n)
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

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			if nodegroupID != "" {
				ng, _, err := mksnodegroup.Get(ctx, mksClient, clusterID, nodegroupID)
				if err != nil {
					return err
				}
				return output.Print(cmd, ng.Nodes)
			}

			groups, _, err := mksnodegroup.List(ctx, mksClient, clusterID)
			if err != nil {
				return err
			}
			nodes := make([]*mksnode.View, 0)
			for _, ng := range groups {
				nodes = append(nodes, ng.Nodes...)
			}
			return output.Print(cmd, nodes)
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVar(&nodegroupID, "nodegroup-id", "", "nodegroup ID")

	return cmd
}
