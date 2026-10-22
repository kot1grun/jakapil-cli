/*
Copyright © 2026 Kot1gruN
*/
package registry

import (
	"context"

	craasregistry "github.com/selectel/craas-go/pkg/v1/registry"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
	"jakapil-cli/internal/output"
)

// newShowCommand returns the "registry show" subcommand that returns a single
// registry by its identifier, backed by the registry.Get SDK function.
func newShowCommand() *cobra.Command {
	var registryID string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show details of a registry",
		Long: `Show details of a registry by its identifier.

Example:
  jakapil-cli craas registry show --id <registry-id>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if registryID == "" {
				return cmd.Help()
			}

			crClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			r, _, err := craasregistry.Get(ctx, crClient, registryID)
			if err != nil {
				return err
			}
			return output.Print(cmd, r)
		},
	}

	cmd.Flags().StringVarP(&registryID, "id", "i", "", "registry ID to show")
	return cmd
}
