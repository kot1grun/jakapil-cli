/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"fmt"
	"os"
	"strings"

	craasclient "github.com/selectel/craas-go/pkg/v1/client"
	craasclientv2 "github.com/selectel/craas-go/pkg/v2/client"
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
// the given command using the resolved token and endpoint.
func NewClient(cmd *cobra.Command) (*craasclient.ServiceClient, error) {
	token, endpoint, err := resolveCredentials(cmd)
	if err != nil {
		return nil, err
	}

	return craasclient.NewCRaaSClientV1(token, endpoint)
}

// NewClientV2 builds an authenticated craas-go ServiceClient for the V2 API
// from the given command. The endpoint is derived from the resolved V1
// endpoint by replacing its trailing version, so "…/api/v1" (or "…/api/v2")
// becomes "…/api/v2".
func NewClientV2(cmd *cobra.Command) (*craasclientv2.ServiceClient, error) {
	token, endpoint, err := resolveCredentials(cmd)
	if err != nil {
		return nil, err
	}
	endpoint = v2Endpoint(endpoint)

	return craasclientv2.NewCRaaSClientV2(token, endpoint)
}

// resolveCredentials returns the token and endpoint resolved from the command
// flags with the environment variables as fallbacks.
func resolveCredentials(cmd *cobra.Command) (string, string, error) {
	token, err := cmd.Flags().GetString(FlagToken)
	if err != nil {
		return "", "", err
	}
	if token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return "", "", fmt.Errorf("a token is required: pass --token or set %s", EnvToken)
	}

	endpoint, err := cmd.Flags().GetString(FlagEndpoint)
	if err != nil {
		return "", "", err
	}
	if endpoint == "" || !cmd.Flags().Changed(FlagEndpoint) {
		if env := os.Getenv(EnvEndpoint); env != "" {
			endpoint = env
		}
	}
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	return token, endpoint, nil
}

// v2Endpoint returns the V2 API endpoint derived from a V1/V2 endpoint by
// replacing its trailing version segment. A trailing slash is trimmed first so
// the result always ends in "/v2".
func v2Endpoint(endpoint string) string {
	base := strings.TrimRight(endpoint, "/")
	base = strings.TrimSuffix(base, "/v1")
	base = strings.TrimSuffix(base, "/v2")
	return base + "/v2"
}
