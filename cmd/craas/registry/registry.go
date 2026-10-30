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

// NewRegistryCommand returns the "registry" subcommand.
// Without arguments it shows the command help; with "list" it lists all
// registries; with "create" it creates one; with "show" or "--id" it returns a
// single registry; with "delete" it removes one. The persistent "--output"
// flag controls the rendering format.
func NewRegistryCommand() *cobra.Command {
	var registryID string

	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage containers registries (list, show, create, delete)",
		Long: `Manage registries of the Selectel Container Registry Service.

Subcommands:
  list    List all registries in the current project
  create  Create a new registry
  show    Show details of a registry by its identifier (uses Get)
  delete  Delete a registry by its identifier

Flags:
  -i, --id  Show details of a single registry by its identifier

Examples:
  jakapil-cli craas registry list
  jakapil-cli craas registry --id <registry-id>`,
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
			return render(cmd, r)
		},
	}

	cmd.Flags().StringVarP(&registryID, "id", "i", "", "registry ID to show details for")
	cmd.AddCommand(
		newListCommand(),
		newShowCommand(),
		newCreateCommand(),
		newDeleteCommand(),
	)

	return cmd
}

// newListCommand returns the "registry list" subcommand.
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registries",
		Long:  "List all registries in the current project.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			crClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			registries, _, err := craasregistry.List(ctx, crClient)
			if err != nil {
				return err
			}
			return render(cmd, registries)
		},
	}
}
