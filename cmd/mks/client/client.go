/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"fmt"
	"os"

	v1 "github.com/selectel/mks-go/pkg/v1"
	"github.com/spf13/cobra"
)

const (
	// FlagToken is the persistent flag name that holds the Selectel Keystone API token.
	FlagToken = "token"

	// FlagEndpoint is the persistent flag name that holds the Selectel Keystone API endpoint.
	FlagEndpoint = "endpoint"

	// EnvToken is the environment variable used as a fallback for FlagToken.
	EnvToken = "SLT_KS_TOKEN"

	// EnvEndpoint is the environment variable used as a fallback for FlagEndpoint.
	EnvEndpoint = "SLT_ENDPOINT"

	// DefaultEndpoint is used when neither flag nor config provides an endpoint.
	DefaultEndpoint = "https://ru-1.mks.selcloud.ru/v1"
)

// NewClient builds a MKS V1 ServiceClient from the given command.
// The token is taken from the "--token" flag or from the SLT_KS_TOKEN
// environment variable; the endpoint is taken from the "--endpoint" flag,
// otherwise from the SLT_ENDPOINT environment variable, or defaults to
// DefaultEndpoint.
func NewClient(cmd *cobra.Command) (*v1.ServiceClient, error) {
	token, err := cmd.Flags().GetString(FlagToken)
	if err != nil || token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return nil, fmt.Errorf("authentication token is required: pass --%s flag or set %s env var", FlagToken, EnvToken)
	}

	endpoint, err := cmd.Flags().GetString(FlagEndpoint)
	// The flag carries a non-empty default value (DefaultEndpoint), yet it must
	// only win over the environment when explicitly set by the user. Otherwise
	// SLT_ENDPOINT would never be honored.
	if endpoint == "" || !cmd.Flags().Changed(FlagEndpoint) {
		endpoint = os.Getenv(EnvEndpoint)
	}
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	return v1.NewMKSClientV1(token, endpoint), nil
}
