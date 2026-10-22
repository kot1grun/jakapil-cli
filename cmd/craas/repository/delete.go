/*
Copyright © 2026 Kot1gruN
*/
package repository

import (
	"context"
	"fmt"
	"net/http"

	craasrepository "github.com/selectel/craas-go/pkg/v1/repository"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
)

// newDeleteCommand returns the "repository delete" subcommand that deletes a
// repository by its name.
//
// The operation is destructive, so it requires an explicit confirmation via
// the "--yes" flag before the request is sent.
func newDeleteCommand() *cobra.Command {
	var (
		registryID     string
		repositoryName string
		confirm        bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a repository",
		Long: `Delete a repository by its name.

This operation is destructive and cannot be undone. Provide the registry ID
and the repository name, and explicitly confirm the deletion with "--yes".

Example:
  jakapil-cli craas repository delete --registry-id <registry-id> --name <repo> --yes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if registryID == "" || repositoryName == "" {
				return cmd.Help()
			}
			if !confirm {
				return fmt.Errorf("deletion of repository %q was not confirmed: pass --yes to proceed", repositoryName)
			}

			crClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			res, err := craasrepository.DeleteRepository(ctx, crClient, registryID, repositoryName)
			if err != nil {
				return err
			}
			if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusAccepted {
				return fmt.Errorf("unexpected HTTP status %d while deleting repository %q", res.StatusCode, repositoryName)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "repository %s deleted\n", repositoryName)
			return err
		},
	}

	cmd.Flags().StringVar(&registryID, "registry-id", "", "registry ID the repository belongs to")
	cmd.Flags().StringVar(&repositoryName, "name", "", "repository name to delete")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "confirm the destructive repository deletion")

	return cmd
}
