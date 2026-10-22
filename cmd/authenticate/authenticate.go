/*
Copyright © 2026 Kot1gruN
*/
package authenticate

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewAuthenticateCommand returns the "authenticate" command.
//
// It is a placeholder: as of now it does not implement any authentication
// flow. The actual logic (obtaining and persisting a Selectel Keystone token)
// is planned and will be added here.
func NewAuthenticateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authenticate",
		Short: "Authenticate to Selectel and obtain an API token",
		Long: `Authenticate to Selectel and obtain an API token.

This command is a placeholder and does not perform any action yet.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "authenticate: not implemented yet")
			return nil
		},
	}

	return cmd
}