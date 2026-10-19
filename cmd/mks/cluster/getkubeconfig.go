/*
Copyright © 2026 Kot1gruN
*/
package cluster

import (
	"context"
	"fmt"
	"os"

	mkscluster "github.com/selectel/mks-go/pkg/v1/cluster"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// newGetKubeconfigCommand returns the "cluster get-kubeconfig" subcommand that
// downloads the kubeconfig of a cluster by its identifier.
//
// The raw kubeconfig is written to stdout by default, or to the file given via
// "--file" (useful for KUBECONFIG copies and automation).
func newGetKubeconfigCommand() *cobra.Command {
	var (
		clusterID string
		filePath  string
	)

	cmd := &cobra.Command{
		Use:   "get-kubeconfig",
		Short: "Get cluster kubeconfig",
		Long: `Get the kubeconfig of a Kubernetes cluster by its identifier.

The raw kubeconfig is printed to stdout by default. Pass "--file" to write it
directly to a file instead (e.g. for KUBECONFIG export).

Examples:
  jakapil-cli mks cluster get-kubeconfig --id <cluster-id>
  jakapil-cli mks cluster get-kubeconfig --id <cluster-id> --file ./kubeconfig`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if clusterID == "" {
				return cmd.Help()
			}

			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			kubeconfig, _, err := mkscluster.GetKubeconfig(ctx, mksClient, clusterID)
			if err != nil {
				return err
			}
			if kubeconfig == nil {
				return fmt.Errorf("got an empty kubeconfig for cluster %q", clusterID)
			}

			if filePath != "" {
				if err := os.WriteFile(filePath, kubeconfig, 0o600); err != nil {
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "kubeconfig for cluster %s written to %s\n", clusterID, filePath)
				return err
			}

			_, err = cmd.OutOrStdout().Write(kubeconfig)
			if err == nil {
				_, err = cmd.OutOrStdout().Write([]byte("\n"))
			}
			return err
		},
	}

	cmd.Flags().StringVar(&clusterID, "id", "", "cluster ID to get the kubeconfig for")
	cmd.Flags().StringVar(&filePath, "file", "", "path to write the kubeconfig to (default: stdout)")

	return cmd
}
