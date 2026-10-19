/*
Copyright © 2026 Kot1gruN
*/
package nodegroup

import (
	"context"
	"fmt"

	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// NewNodeGroupCommand returns the "nodegroup" subcommand.
// Without "--id" it lists all nodegroups of a cluster; with "--id" it
// returns a single nodegroup referenced by its identifier.
func NewNodeGroupCommand() *cobra.Command {
	var (
		clusterID string
		groupID   string
	)

	cmd := &cobra.Command{
		Use:   "nodegroup",
		Short: "List cluster nodegroups or get a single nodegroup by ID",
		Long: `List all nodegroups of a Kubernetes cluster, or get details
of a single nodegroup by its identifier.

Examples:
  jakapil-cli mks nodegroup --cluster-id <cluster-id>
  jakapil-cli mks nodegroup --cluster-id <cluster-id> --id <nodegroup-id>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			if groupID != "" {
				ng, _, err := mksnodegroup.Get(ctx, mksClient, clusterID, groupID)
				if err != nil {
					return err
				}
				fmt.Printf("%+v\n", ng)
				return nil
			}

			groups, _, err := mksnodegroup.List(ctx, mksClient, clusterID)
			if err != nil {
				return err
			}
			for _, ng := range groups {
				fmt.Printf("%+v\n", ng)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVarP(&groupID, "id", "i", "", "nodegroup ID to get details for")

	return cmd
}
