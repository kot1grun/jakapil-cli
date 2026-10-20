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
	"jakapil-cli/cmd/mks/output"
)

// NewNodeGroupCommand returns the "nodegroup" subcommand.
// Without arguments it shows the command help; "list" lists nodegroups of a
// cluster; with "--cluster-id" and "--id" it returns a single nodegroup.
func NewNodeGroupCommand() *cobra.Command {
	var (
		clusterID string
		groupID   string
	)

	cmd := &cobra.Command{
		Use:   "nodegroup",
		Short: "Manage cluster nodegroups (list, get)",
		Long: `Manage nodegroups of a Kubernetes cluster.

Subcommands:
  list    List nodegroups of a cluster
  resize  Resize a nodegroup by its desired node count
  delete  Delete a nodegroup by its identifier

Flags:
  -i, --id  Get details of a single nodegroup

Examples:
  jakapil-cli mks nodegroup list --cluster-id <cluster-id>
  jakapil-cli mks nodegroup --cluster-id <cluster-id> --id <nodegroup-id>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" || groupID == "" {
				return cmd.Help()
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			ng, _, err := mksnodegroup.Get(ctx, mksClient, clusterID, groupID)
			if err != nil {
				return err
			}
			return output.Print(cmd, ng)
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVarP(&groupID, "id", "i", "", "nodegroup ID to get details for")
	cmd.AddCommand(newListCommand(), newResizeCommand(), newDeleteCommand())

	return cmd
}

// newListCommand returns the "nodegroup list" subcommand.
func newListCommand() *cobra.Command {
	var clusterID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List nodegroups of a cluster",
		Long:  "List all nodegroups of a Kubernetes cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			groups, _, err := mksnodegroup.List(ctx, mksClient, clusterID)
			if err != nil {
				return err
			}
			return output.Print(cmd, groups)
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")

	return cmd
}
