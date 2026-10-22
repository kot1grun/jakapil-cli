/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"fmt"
	"os"

	craasclient "github.com/selectel/craas-go/pkg/v1/client"
	"github.com/spf13/cobra"
)

const (
	// FlagToken is the persistent flag name that holds the Selectel API token
	// used to authenticate to the CRaaS API.
	FlagToken = "token"

	// FlagEndpoint is the persistent flag name that holds the CRaaS API
	// endpoint, e.g. https://cr.selcloud.ru/api/v1.
	FlagEndpoint = "endpoint"

	// EnvToken is the environment variable used as a fallback for FlagToken.
	EnvToken = "SLT_KS_TOKEN"

	// EnvEndpoint is the environment variable used as a fallback for
	// FlagEndpoint.
	EnvEndpoint = "SLT_CR_ENDPOINT"

	// DefaultEndpoint is the CRaaS API endpoint used when neither the flag nor
	// the environment variable is set.
	DefaultEndpoint = "https://cr.selcloud.ru/api/v1"
)

// NewClient builds an authenticated craas-go ServiceClient for the V1 API from
// the given command. The token is taken from the "--token" flag or the
// SLT_KS_TOKEN environment variable; the endpoint from the "--endpoint" flag,
// otherwise from the SLT_CR_ENDPOINT environment variable, or DefaultEndpoint.
func NewClient(cmd *cobra.Command) (*craasclient.ServiceClient, error) {
	token, err := cmd.Flags().GetString(FlagToken)
	if err != nil {
		return nil, err
	}
	if token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return nil, fmt.Errorf("a token is required: pass --token or set %s", EnvToken)
	}

	endpoint, err := cmd.Flags().GetString(FlagEndpoint)
	if err != nil {
		return nil, err
	}
	if endpoint == "" || !cmd.Flags().Changed(FlagEndpoint) {
		if env := os.Getenv(EnvEndpoint); env != "" {
			endpoint = env
		}
	}
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	return craasclient.NewCRaaSClientV1(token, endpoint)
}
