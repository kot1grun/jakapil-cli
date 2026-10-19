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

// newRotateCertsCommand returns the "cluster rotate-certs" subcommand that
// requests a rotation of the cluster's certificates by its identifier.
//
// The operation changes the state of the cluster, so it requires an explicit
// confirmation via the "--yes" flag before the request is sent.
func newRotateCertsCommand() *cobra.Command {
	var (
		clusterID string
		confirm   bool
	)

	cmd := &cobra.Command{
		Use:   "rotate-certs",
		Short: "Rotate cluster certificates",
		Long: `Request the rotation of certificates for a Kubernetes cluster by its
identifier.

This operation re-issues the cluster certificates and may restart cluster
components, so it requires an explicit confirmation with "--yes".

Example:
  jakapil-cli mks cluster rotate-certs --id <cluster-id> --yes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return cmd.Help()
			}
			if !confirm {
				return fmt.Errorf("certificate rotation for cluster %q was not confirmed: pass --yes to proceed", clusterID)
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			res, err := mkscluster.RotateCerts(ctx, mksClient, clusterID)
			if err != nil {
				return err
			}
			// The rotation is async and commonly returns 202 Accepted or 204 No
			// Content; treat any 2xx status as success.
			if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
				return fmt.Errorf("unexpected HTTP status %d while rotating certificates for cluster %q", res.StatusCode, clusterID)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "certificate rotation for cluster %s requested\n", clusterID)
			return err
		},
	}

	cmd.Flags().StringVar(&clusterID, "id", "", "cluster ID to rotate certificates for")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "confirm the certificate rotation")

	return cmd
}
