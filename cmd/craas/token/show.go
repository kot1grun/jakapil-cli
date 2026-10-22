/*
Copyright © 2026 Kot1gruN
*/
package token

import (
	"context"

	craastoken "github.com/selectel/craas-go/pkg/v1/token"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
	"jakapil-cli/cmd/mks/output"
)

// newShowCommand returns the "token show" subcommand that returns a single
// token by its identifier, backed by the token.Get SDK function.
func newShowCommand() *cobra.Command {
	var tokenID string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show details of a token",
		Long: `Show details of a token by its identifier.

Example:
  jakapil-cli craas token show --id <token-id>`,
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

			t, _, err := craastoken.Get(ctx, crClient, tokenID)
			if err != nil {
				return err
			}
			return output.Print(cmd, t)
		},
	}

	cmd.Flags().StringVarP(&tokenID, "id", "i", "", "token ID to show")
	return cmd
}
