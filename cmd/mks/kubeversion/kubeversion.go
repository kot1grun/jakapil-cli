/*
Copyright © 2026 Kot1gruN
*/
package kubeversion

import (
	"context"
	"fmt"

	mkskubeversion "github.com/selectel/mks-go/pkg/v1/kubeversion"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
)

// NewKubeVersionCommand returns the "kubeversion" subcommand.
// It lists all Kubernetes versions supported by Selectel MKS.
func NewKubeVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "kubeversion",
		Short: "List supported Kubernetes versions",
		Long: `List all Kubernetes versions supported by the Selectel Managed
Kubernetes Service.

Example:
  jakapil-cli mks kubeversion`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mksClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			versions, _, err := mkskubeversion.List(ctx, mksClient)
			if err != nil {
				return err
			}
			for _, v := range versions {
				fmt.Printf("version=%s default=%t\n", v.Version, v.IsDefault)
			}

			return nil
		},
	}
}
