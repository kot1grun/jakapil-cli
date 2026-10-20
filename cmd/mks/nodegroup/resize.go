/*
Copyright © 2026 Kot1gruN
*/
package nodegroup

import (
	"context"
	"fmt"
	"net/http"

	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// newResizeCommand returns the "nodegroup resize" subcommand that resizes a
// nodegroup of a cluster by setting the desired number of nodes.
func newResizeCommand() *cobra.Command {
	var (
		clusterID string
		groupID   string
		desired   int
	)

	cmd := &cobra.Command{
		Use:   "resize",
		Short: "Resize a nodegroup",
		Long: `Resize a nodegroup of a Kubernetes cluster by setting the desired
number of nodes.

The operation changes the state of the cluster and may create or remove nodes.

Examples:
  jakapil-cli mks nodegroup resize --cluster-id <cluster-id> --id <nodegroup-id> --desired 3`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}
			if groupID == "" {
				return cmd.Help()
			}
			if desired <= 0 {
				return fmt.Errorf("--desired must be a positive integer")
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			opts := &mksnodegroup.ResizeOpts{Desired: desired}
			res, err := mksnodegroup.Resize(ctx, mksClient, clusterID, groupID, opts)
			if err != nil {
				return err
			}
			// The resize is async and commonly returns 202 Accepted or 204 No
			// Content; treat any 2xx status as success.
			if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("unexpected HTTP status %d while resizing nodegroup %q", res.StatusCode, groupID)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "request to resize %s nodegroup to %d node(s) scheduled\n", groupID, desired)
			return err
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVarP(&groupID, "id", "i", "", "nodegroup ID to resize")
	cmd.Flags().IntVar(&desired, "desired", 0, "desired number of nodes in the nodegroup")

	return cmd
}
