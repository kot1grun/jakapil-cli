/*
Copyright © 2026 Kot1gruN
*/
package token

import (
	"context"
	"fmt"
	"net/http"

	craastoken "github.com/selectel/craas-go/pkg/v1/token"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
)

// newDeleteCommand returns the "token revoke" subcommand that revokes a token
// by its identifier, backed by the token.Revoke SDK function.
//
// The operation is destructive, so it requires an explicit confirmation via
// the "--yes" flag before the request is sent.
func newDeleteCommand() *cobra.Command {
	var (
		tokenID string
		confirm bool
	)

	cmd := &cobra.Command{
		Use:     "revoke",
		Aliases: []string{"delete"},
		Short:   "Revoke (delete) a token",
		Long: `Revoke a token by its identifier.

This operation is destructive and cannot be undone. Provide the token ID with
"--id" and explicitly confirm the revocation with "--yes".

Example:
  jakapil-cli craas token revoke --id <token-id> --yes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if tokenID == "" {
				return cmd.Help()
			}
			if !confirm {
				return fmt.Errorf("revocation of token %q was not confirmed: pass --yes to proceed", tokenID)
			}

			crClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			res, err := craastoken.Revoke(ctx, crClient, tokenID)
			if err != nil {
				return err
			}
			if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusAccepted {
				return fmt.Errorf("unexpected HTTP status %d while revoking token %q", res.StatusCode, tokenID)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "token %s revoked\n", tokenID)
			return err
		},
	}

	cmd.Flags().StringVarP(&tokenID, "id", "i", "", "token ID to revoke")
	cmd.Flags().BoolVarP(&confirm, "yes", "y", false, "confirm the destructive token revocation")

	return cmd
}
