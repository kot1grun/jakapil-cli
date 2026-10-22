/*
Copyright © 2026 Kot1gruN
*/
package token

import (
	"github.com/spf13/cobra"
)

// NewTokenCommand returns the "token" subcommand.
// It provides CRUD operations over container registry API tokens: "create",
// "show" (read), "refresh" (update) and "revoke" (delete). The persistent
// "--output" flag controls the rendering format.
func NewTokenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage container registry API tokens (create, list, show, refresh, revoke)",
		Long: `Manage API tokens of the Selectel Container Registry Service.

Subcommands:
  create  Create a new token
  list    List API tokens (uses the V2 API)
  show    Show details of a token by its identifier (uses Get)
  refresh Refresh a token by its identifier (uses Refresh)
  revoke  Revoke a token by its identifier (uses Revoke)

Note: listing tokens is only provided by the V2 API (GET /api/v2/tokens).

Examples:
  jakapil-cli craas token create --ttl 1y
  jakapil-cli craas token list --limit 50
  jakapil-cli craas token show --id <token-id>`,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(
		newCreateCommand(),
		newListCommand(),
		newShowCommand(),
		newRefreshCommand(),
		newDeleteCommand(),
	)

	return cmd
}
