/*
Copyright © 2026 Kot1gruN
*/
package kubeversion

import (
	"context"

	mkskubeversion "github.com/selectel/mks-go/pkg/v1/kubeversion"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/mks/client"
	"jakapil-cli/cmd/mks/output"
)

// NewKubeVersionCommand returns the "kubeversion" subcommand.
// Without arguments it shows the command help; "list" lists all Kubernetes
// versions supported by Selectel MKS.
func NewKubeVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeversion",
		Short: "Show supported Kubernetes versions",
		Long: `Show Kubernetes versions supported by the Selectel Managed
Kubernetes Service.

Subcommands:
  list  List all supported Kubernetes versions

Example:
  jakapil-cli mks kubeversion list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newListCommand())

	return cmd
}

// newListCommand returns the "kubeversion list" subcommand.
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List supported Kubernetes versions",
		Long:  "List all Kubernetes versions supported by the Selectel Managed Kubernetes Service.",
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
			return output.Print(cmd, versions)
		},
	}
}
