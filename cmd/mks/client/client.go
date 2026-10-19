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
	// FlagToken is the persistent flag name that holds the MKS API token.
	FlagToken = "token"

	// FlagEndpoint is the persistent flag name that holds the MKS API endpoint.
	FlagEndpoint = "endpoint"

	// EnvToken is the environment variable used as a fallback for FlagToken.
	EnvToken = "MKS_TOKEN"

	// DefaultEndpoint is used when neither flag nor config provides an endpoint.
	DefaultEndpoint = "https://ru-1.mks.selcloud.ru/v1"
)

// NewClient builds a MKS V1 ServiceClient from the given command.
// The token is taken from the "--token" flag or from the MKS_TOKEN
// environment variable; the endpoint is taken from the "--endpoint" flag
// or defaults to DefaultEndpoint.
func NewClient(cmd *cobra.Command) (*v1.ServiceClient, error) {
	token, err := cmd.Flags().GetString(FlagToken)
	if err != nil || token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return nil, fmt.Errorf("authentication token is required: pass --%s flag or set %s env var", FlagToken, EnvToken)
	}

	endpoint, err := cmd.Flags().GetString(FlagEndpoint)
	if err != nil || endpoint == "" {
		endpoint = DefaultEndpoint
	}

	return v1.NewMKSClientV1(token, endpoint), nil
}
