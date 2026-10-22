/*
Copyright © 2026 Kot1gruN
*/
package repository

import (
	"context"

	craasrepository "github.com/selectel/craas-go/pkg/v1/repository"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
	"jakapil-cli/cmd/mks/output"
)

// newShowCommand returns the "repository show" subcommand that returns a single
// repository by its name, backed by the repository.GetRepository SDK function.
func newShowCommand() *cobra.Command {
	var (
		registryID     string
		repositoryName string
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show details of a repository",
		Long: `Show details of a repository by its name.

Example:
  jakapil-cli craas repository show --registry-id <registry-id> --name <repo>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if registryID == "" || repositoryName == "" {
				return cmd.Help()
			}

			crClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			repo, _, err := craasrepository.GetRepository(ctx, crClient, registryID, repositoryName)
			if err != nil {
				return err
			}
			return output.Print(cmd, repo)
		},
	}

	cmd.Flags().StringVar(&registryID, "registry-id", "", "registry ID the repository belongs to")
	cmd.Flags().StringVar(&repositoryName, "name", "", "repository name to show")
	return cmd
}
