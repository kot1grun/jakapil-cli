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

// newDeleteCommand returns the "nodegroup delete" subcommand that deletes a
// nodegroup of a cluster by its identifier.
//
// The operation is destructive, so it requires an explicit confirmation via
// the "--yes" flag before the request is sent.
func newDeleteCommand() *cobra.Command {
	var (
		clusterID string
		groupID   string
		confirm   bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a nodegroup",
		Long: `Delete a nodegroup of a Kubernetes cluster by its identifier.

This operation is destructive and cannot be undone. Provide the cluster ID
with "--cluster-id" and the nodegroup ID with "--id", and explicitly confirm
the deletion with "--yes".

Example:
  jakapil-cli mks nodegroup delete --cluster-id <cluster-id> --id <nodegroup-id> --yes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return fmt.Errorf("--cluster-id is required")
			}
			if groupID == "" {
				return cmd.Help()
			}
			if !confirm {
				return fmt.Errorf("deletion of nodegroup %q was not confirmed: pass --yes to proceed", groupID)
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			res, err := mksnodegroup.Delete(ctx, mksClient, clusterID, groupID)
			if err != nil {
				return err
			}
			// Delete commonly returns 204 No Content (or 202 Accepted for the
			// async operation); treat any 2xx status as success.
			if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("unexpected HTTP status %d while deleting nodegroup %q", res.StatusCode, groupID)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "nodegroup %s deleted\n", groupID)
			return err
		},
	}

	cmd.Flags().StringVar(&clusterID, "cluster-id", "", "cluster ID")
	cmd.Flags().StringVarP(&groupID, "id", "i", "", "nodegroup ID to delete")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "confirm the destructive nodegroup deletion")

	return cmd
}
