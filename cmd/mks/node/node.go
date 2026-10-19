/*
Copyright © 2026 Kot1gruN
*/
package node

import (
	"context"
	"fmt"

	mksnode "github.com/selectel/mks-go/pkg/v1/node"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// NewNodeCommand returns the "node" subcommand.
// It returns details of a single node of a cluster nodegroup by its id.
func NewNodeCommand() *cobra.Command {
	var (
		clusterID   string
		nodegroupID string
		nodeID      string
	)

	cmd := &cobra.Command{
		Use:   "node",
		Short: "Get a single node of a cluster nodegroup by ID",
		Long: `Get details of a single Kubernetes node of a cluster nodegroup
by its identifier.

Example:
  jakapil-cli mks node --cluster-id <cluster-id> --nodegroup-id <nodegroup-id> --id <node-id>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}
			if nodegroupID == "" {
				return fmt.Errorf("--nodegroup-id is required")
			}
			if nodeID == "" {
				return fmt.Errorf("--id is required")
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
			fmt.Printf("%+v\n", n)

			return nil
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVar(&nodegroupID, "nodegroup-id", "", "nodegroup ID")
	cmd.Flags().StringVarP(&nodeID, "id", "i", "", "node ID")

	return cmd
}
