/*
Copyright © 2026 Kot1gruN
*/
package token

import (
	"context"

	craastokenv2 "github.com/selectel/craas-go/pkg/v2/token"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/craas/client"
	"jakapil-cli/internal/output"
)

// newListCommand returns the "token list" subcommand that lists API tokens.
// Listing is only available through the V2 API (GET /api/v2/tokens), so it
// uses the V2 client and the tokenv2.List SDK function.
func newListCommand() *cobra.Command {
	var opts listOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API tokens",
		Long: `List container registry API tokens.

Listing uses the V2 API and supports pagination and filtering:

  --limit         maximum number of tokens per page
  --offset        number of tokens to skip
  --sort-field    sort field (e.g. created_at)
  --sort-type     sort order: asc or desc
  --search        full-text search over token fields
  --scope-mode    filter by token scope mode

Example:
  jakapil-cli craas token list --limit 50 --offset 0 --sort-type desc`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			crClient, err := client.NewClientV2(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			tokens, _, err := craastokenv2.List(ctx, crClient, opts.v2Opts())
			if err != nil {
				return err
			}
			return output.Print(cmd, tokens)
		},
	}

	cmd.Flags().IntVar(&opts.Limit, "limit", 0, "maximum number of tokens per page (omit for server default)")
	cmd.Flags().IntVar(&opts.Offset, "offset", 0, "number of tokens to skip")
	cmd.Flags().StringVar(&opts.SortField, "sort-field", "", "sort field, e.g. created_at")
	cmd.Flags().StringVar(&opts.SortType, "sort-type", "", "sort order: asc or desc")
	cmd.Flags().StringVar(&opts.Search, "search", "", "full-text search over token fields")
	cmd.Flags().StringVar(&opts.ScopeMode, "scope-mode", "", "filter by token scope mode")
	return cmd
}

// listOptions holds the flag values for listing tokens.
type listOptions struct {
	Limit     int
	Offset    int
	SortField string
	SortType  string
	Search    string
	ScopeMode string
}

// v2Opts converts the flag values into the craas-go V2 token Opts. Zero values
// for pagination are left as nil so the SDK omits the query parameters.
func (o *listOptions) v2Opts() craastokenv2.Opts {
	opts := craastokenv2.Opts{
		SortField: o.SortField,
		SortType:  o.SortType,
		Search:    o.Search,
		ScopeMode: o.ScopeMode,
	}
	if o.Limit > 0 {
		limit := o.Limit
		opts.Limit = &limit
	}
	if o.Offset > 0 {
		offset := o.Offset
		opts.Offset = &offset
	}
	return opts
}
