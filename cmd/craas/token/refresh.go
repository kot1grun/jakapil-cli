/*
Copyright © 2026 Kot1gruN
*/
package token

import (
	"context"

	craastoken "github.com/selectel/craas-go/pkg/v1/token"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
	"jakapil-cli/internal/output"
)

// newRefreshCommand returns the "token refresh" subcommand that refreshes a
// token by its identifier, backed by the token.Refresh SDK function.
func newRefreshCommand() *cobra.Command {
	var tokenID string

	cmd := &cobra.Command{
		Use:   "refresh",
		Short: "Refresh a token",
		Long: `Refresh a token by its identifier.

Example:
  jakapil-cli craas token refresh --id <token-id>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if tokenID == "" {
				return cmd.Help()
			}

			crClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			t, _, err := craastoken.Refresh(ctx, crClient, tokenID)
			if err != nil {
				return err
			}
			return output.Print(cmd, t)
		},
	}

	cmd.Flags().StringVarP(&tokenID, "id", "i", "", "token ID to refresh")
	return cmd
}
