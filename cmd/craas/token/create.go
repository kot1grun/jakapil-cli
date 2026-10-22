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

// newCreateCommand returns the "token create" subcommand that requests a new
// token with the given TTL.
func newCreateCommand() *cobra.Command {
	var ttl string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new token",
		Long: `Create a new container registry API token.

The token TTL defaults to 12h and can be set to 1y with "--ttl".

Example:
  jakapil-cli craas token create --ttl 1y`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			crClient, err := client.NewClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			opts := &craastoken.CreateOpts{TokenTTL: craastoken.TTL(ttl)}
			t, _, err := craastoken.Create(ctx, crClient, opts)
			if err != nil {
				return err
			}
			return output.Print(cmd, t)
		},
	}

	cmd.Flags().StringVar(&ttl, "ttl", string(craastoken.TTL12Hours), "token TTL: 12h or 1y")
	return cmd
}
