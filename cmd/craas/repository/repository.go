/*
Copyright © 2026 Kot1gruN
*/
package repository

import (
	"context"

	craasrepository "github.com/selectel/craas-go/pkg/v1/repository"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
)

// NewRepositoryCommand returns the "repository" subcommand.
// It manages repositories of a given registry: "list" lists all repositories,
// "show" returns a single one, "delete" removes one. The persistent "--output"
// flag controls the rendering format.
func NewRepositoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repository",
		Short: "Manage container repositories (list, show, delete)",
		Long: `Manage repositories of the Selectel Container Registry Service.

Subcommands:
  list    List all repositories in a registry
  show    Show details of a repository by its identifier (uses GetRepository)
  delete  Delete a repository by its identifier

Repositories are created implicitly when the first image is pushed; there is no
explicit "create" operation provided by the API.

Examples:
  jakapil-cli craas repository list --registry-id <registry-id>
  jakapil-cli craas repository show --registry-id <registry-id> --name <repo>`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
		newShowCommand(),
		newDeleteCommand(),
	)

	return cmd
}

// newListCommand returns the "repository list" subcommand.
func newListCommand() *cobra.Command {
	var registryID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all repositories in a registry",
		Long: `List all repositories in the given registry.

Example:
  jakapil-cli craas repository list --registry-id <registry-id>`,
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

			repos, _, err := craasrepository.ListRepositories(ctx, crClient, registryID)
			if err != nil {
				return err
			}
			return render(cmd, repos)
		},
	}

	cmd.Flags().StringVar(&registryID, "registry-id", "", "registry ID to list repositories of")
	return cmd
}
