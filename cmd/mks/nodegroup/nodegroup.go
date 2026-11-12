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
	"jakapil-cli/internal/output"
	"jakapil-cli/internal/parallel"
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
		Short: "Manage cluster nodegroups",
		Long: `Manage nodegroups of a Kubernetes cluster.

Subcommands:
  list    List nodegroups of a cluster
  create  Create a nodegroup in a cluster
  update  Update nodegroup labels, taints or autoscaling
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
			return render(cmd, ng)
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVarP(&groupID, "id", "i", "", "nodegroup ID to get details for")
	cmd.AddCommand(newListCommand(), newResizeCommand(), newDeleteCommand(), newCreateCommand(), newUpdateCommand())

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
			if allRegions, _ := cmd.Flags().GetBool(client.FlagAllRegions); allRegions {
				return runListAllRegions(cmd, clusterID)
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
			if len(groups) == 0 && output.IsTable(cmd) {
				return output.PrintListEmpty(cmd, client.EffectiveRegion(cmd, mksClient))
			}
			return render(cmd, groups)
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().Bool(client.FlagAllRegions, false, "list nodegroups in every region the service advertises instead of --region")

	return cmd
}

// runListAllRegions lists nodegroups of a cluster in every region the service
// advertises and renders the aggregated result with a "Region" column (table)
// or grouped by region (json/yaml), backed by the --all-regions flag. Regions
// are queried in parallel with parallel.Regions, since the underlying calls are
// read-only GETs. A region whose endpoint is unavailable is skipped with a
// warning on stderr; the remaining regions are still rendered.
func runListAllRegions(cmd *cobra.Command, clusterID string) error {
	regions, clients, err := client.NewClients(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()

	results := parallel.Regions(ctx, regions, func(ctx context.Context, region string) ([][]string, interface{}, error) {
		groups, _, err := mksnodegroup.List(ctx, clients[region], clusterID)
		if err != nil {
			return nil, nil, err
		}
		rows := make([][]string, 0, len(groups))
		for _, ng := range groups {
			rows = append(rows, nodegroupListViewRow(ng))
		}
		return rows, groups, nil
	})

	return output.RenderRegionsBestEffort(cmd, nodegroupHeader, results)
}
