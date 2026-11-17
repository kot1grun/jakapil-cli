/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"errors"
	"fmt"
	"os"

	v1 "github.com/selectel/globalrouter-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/authentication"
)

const (
	// FlagToken is the persistent flag name that holds the Selectel Keystone API token.
	FlagToken = "token"

	// EnvToken is the environment variable used as a fallback for FlagToken.
	EnvToken = "SLT_KS_TOKEN"

	// DefaultAPIUrl is the fixed Global Router (NaaS) v1 API endpoint. Unlike the
	// other Selectel SDK services in this repo, the Global Router API does not
	// advertise itself in the Keystone service catalog, so its endpoint is
	// hard-coded instead of being resolved through discovery.
	DefaultAPIUrl = "https://api.selectel.ru/naas/v1"
)

// NewClient builds a Global Router V1 ServiceClient from the given command.
//
// The token is taken from the "--token" flag, otherwise from the persisted
// stored token (account scope, the same one Resell/selproject consume) or the
// SLT_KS_TOKEN environment variable. Service discovery is not needed: the
// client is pinned to the fixed DefaultAPIUrl endpoint.
func NewClient(cmd *cobra.Command) (*v1.ServiceClient, error) {
	token, err := resolveToken(cmd)
	if err != nil {
		return nil, err
	}
	return v1.NewClientV1(token, v1.WithAPIUrl(DefaultAPIUrl))
}

// resolveToken returns the API token from the "--token" flag, falling back to
// the persisted stored token (when it has not expired) and then to the
// SLT_KS_TOKEN environment variable.
func resolveToken(cmd *cobra.Command) (string, error) {
	token, err := cmd.Flags().GetString(FlagToken)
	if err == nil && token == "" {
		var storedErr error
		// The Global Router API requires an account-scoped token (issued with
		// "authenticate token issue --account-scope"), which is not what
		// LoadUsableStoredToken returns: that helper yields the project token
		// consumed by MKS/CRaaS. Use the account-scoped loader instead.
		token, storedErr = authentication.LoadUsableAccountToken()
		if storedErr != nil {
			if errors.Is(storedErr, authentication.ErrTokenExpired) {
				return "", storedErr
			}
			// A non-expiry read problem falls back to the environment.
			token = ""
		}
	}
	if err != nil || token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return "", fmt.Errorf("account-scoped authentication token is required: pass --%s, run \"jakapil-cli authenticate token issue --account-scope\", or set %s env var", FlagToken, EnvToken)
	}
	return token, nil
}
