/*
Copyright © 2026 Kot1gruN
*/
package registry

import (
	"context"
	"fmt"
	"net/http"

	craasregistry "github.com/selectel/craas-go/pkg/v1/registry"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
)

// newDeleteCommand returns the "registry delete" subcommand that deletes a
// registry by its identifier.
//
// The operation is destructive, so it requires an explicit confirmation via
// the "--yes" flag before the request is sent.
func newDeleteCommand() *cobra.Command {
	var (
		registryID string
		confirm    bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a registry",
		Long: `Delete a registry by its identifier.

This operation is destructive and cannot be undone. Provide the registry ID
with "--id" and explicitly confirm the deletion with "--yes".

Example:
  jakapil-cli craas registry delete --id <registry-id> --yes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if registryID == "" {
				return cmd.Help()
			}
			if !confirm {
				return fmt.Errorf("deletion of registry %q was not confirmed: pass --yes to proceed", registryID)
			}

			crClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			res, err := craasregistry.Delete(ctx, crClient, registryID)
			if err != nil {
				return err
			}
			if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusAccepted {
				return fmt.Errorf("unexpected HTTP status %d while deleting registry %q", res.StatusCode, registryID)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "registry %s deleted\n", registryID)
			return err
		},
	}

	cmd.Flags().StringVar(&registryID, "id", "", "registry ID to delete")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "confirm the destructive registry deletion")

	return cmd
}
