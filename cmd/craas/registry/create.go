/*
Copyright © 2026 Kot1gruN
*/
package registry

import (
	"context"

	craasregistry "github.com/selectel/craas-go/pkg/v1/registry"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
)

// newCreateCommand returns the "registry create" subcommand that creates a new
// registry with the given name.
func newCreateCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new registry",
		Long: `Create a new registry in the Selectel Container Registry Service.

The registry name is required.

Example:
  jakapil-cli craas registry create --name my-registry`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return cmd.Help()
			}

			crClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			r, _, err := craasregistry.Create(ctx, crClient, name)
			if err != nil {
				return err
			}
			return render(cmd, r)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "registry name")
	return cmd
}
