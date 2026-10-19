/*
Copyright © 2026 Kot1gruN
*/
package cluster

import (
	"context"
	"fmt"
	"net/http"

	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// newDeleteCommand returns the "cluster delete" subcommand that deletes a
// Kubernetes cluster by its identifier.
//
// The operation is destructive, so it requires an explicit confirmation via
// the "--yes" flag before the request is sent.
func newDeleteCommand() *cobra.Command {
	var (
		clusterID string
		confirm   bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a cluster",
		Long: `Delete a Kubernetes cluster by its identifier.

This operation is destructive and cannot be undone. Provide the cluster ID
with "--id" and explicitly confirm the deletion with "--yes".

Example:
  jakapil-cli mks cluster delete --id <cluster-id> --yes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return cmd.Help()
			}
			if !confirm {
				return fmt.Errorf("deletion of cluster %q was not confirmed: pass --yes to proceed", clusterID)
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			res, err := mkscluster.Delete(ctx, mksClient, clusterID)
			if err != nil {
				return err
			}
			if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusAccepted {
				return fmt.Errorf("unexpected HTTP status %d while deleting cluster %q", res.StatusCode, clusterID)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "cluster %s deleted\n", clusterID)
			return err
		},
	}

	cmd.Flags().StringVar(&clusterID, "id", "", "cluster ID to delete")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "confirm the destructive cluster deletion")

	return cmd
}
