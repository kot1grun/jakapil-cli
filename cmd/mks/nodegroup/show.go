/*
Copyright © 2026 Kot1gruN
*/
package nodegroup

import (
	"context"
	"fmt"

	v1 "github.com/selectel/mks-go/pkg/v1"
	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// newShowCommand returns the "nodegroup show" subcommand that renders the full
// details of a single nodegroup of a cluster. In table mode it prints a vertical
// Parameter/Value table; json/yaml serialize the raw GetView value. nodegroup.Get
// needs the owning cluster ID, so --cluster-id is required (same as list).
func newShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <nodegroup-id>",
		Short: "Show details of a nodegroup",
		Long: `Show the full information about a nodegroup by its ID.

--cluster-id is required: the MKS API resolves a nodegroup only within the context
of its cluster.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clusterID, _ := cmd.Flags().GetString("cluster-id")
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}
			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			return runShow(cmd, mksClient, clusterID, args[0])
		},
	}
	cmd.Flags().String("cluster-id", "", "cluster ID the nodegroup belongs to (required)")
	return cmd
}

// runShow fetches a single nodegroup by its ID within the given cluster and
// renders it through the --output flag.
func runShow(cmd *cobra.Command, mksClient *v1.ServiceClient, clusterID, groupID string) error {
	ctx := context.Background()
	ng, _, err := mksnodegroup.Get(ctx, mksClient, clusterID, groupID)
	if err != nil {
		return err
	}
	return renderShow(cmd, ng)
}
