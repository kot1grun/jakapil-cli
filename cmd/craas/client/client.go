/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"errors"
	"fmt"
	"io"
	"os"

	craasclient "github.com/selectel/craas-go/pkg/v1/client"
	craasclientv2 "github.com/selectel/craas-go/pkg/v2/client"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/authentication"
	"jakapil-cli/internal/keystone"
)

const (
	// FlagToken is the persistent flag name that holds the Selectel Keystone API token.
	FlagToken = "token"

	// FlagEndpoint is the persistent flag name that holds the Selectel Keystone
	// Identity v3 auth URL used for authentication and service discovery.
	FlagEndpoint = "auth-url"

	// FlagDebug is the persistent flag name that enables debug output of the
	// discovered service catalog and CRaaS endpoints.
	FlagDebug = "debug"

	// EnvToken is the environment variable used as a fallback for FlagToken.
	EnvToken = "SLT_KS_TOKEN"

	// EnvEndpoint is the environment variable used as a fallback for FlagEndpoint.
	// It follows the openrc/Keystone convention (OS_AUTH_URL) so that a sourced
	// openrc file is reused instead of a Selectel-specific name.
	EnvEndpoint = "OS_AUTH_URL"

	// DefaultEndpoint is the Selectel Keystone Identity v3 endpoint used when
	// neither the flag nor the environment provides one. The Container Registry
	// API endpoint is then discovered from the service catalog.
	DefaultEndpoint = "https://cloud.api.selcloud.ru/identity/v3"

	// craasServiceType is the CRaaS V1 service type as it appears in the
	// Selectel service catalog.
	craasServiceType = "container-registry"

	// craasServiceTypeV2 is the CRaaS V2 service type as it appears in the
	// Selectel service catalog.
	craasServiceTypeV2 = "container-registry-v2"
)

// NewClient builds an authenticated craas-go ServiceClient for the V1 API from
// the given command. Instead of hitting a hard-coded CRaaS URL, it authenticates
// against the Keystone Identity v3 endpoint (--auth-url or $OS_AUTH_URL) with
// the provided token and resolves the CRaaS V1 API endpoint (service type
// "container-registry") from the returned service catalog.
// The token is taken from the "--token" flag or the SLT_KS_TOKEN environment
// variable; the identity endpoint from the "--auth-url" flag, otherwise from
// the $OS_AUTH_URL environment variable, or DefaultEndpoint.
func NewClient(cmd *cobra.Command) (*craasclient.ServiceClient, error) {
	token, identityEndpoint, debug, err := resolveCredentials(cmd)
	if err != nil {
		return nil, err
	}

	endpoint, err := discoverCRaasEndpoint(identityEndpoint, token, craasServiceType, debug)
	if err != nil {
		return nil, err
	}

	return craasclient.NewCRaaSClientV1(token, endpoint)
}

// NewClientV2 builds an authenticated craas-go ServiceClient for the V2 API
// from the given command. Like NewClient it authenticates against Keystone and
// discovers the CRaaS V2 API endpoint (service type "container-registry-v2") from
// the service catalog.
func NewClientV2(cmd *cobra.Command) (*craasclientv2.ServiceClient, error) {
	token, identityEndpoint, debug, err := resolveCredentials(cmd)
	if err != nil {
		return nil, err
	}

	endpoint, err := discoverCRaasEndpoint(identityEndpoint, token, craasServiceTypeV2, debug)
	if err != nil {
		return nil, err
	}

	return craasclientv2.NewCRaaSClientV2(token, endpoint)
}

// resolveCredentials returns the token, identity endpoint and debug writer
// resolved from the command flags with the environment variables as fallbacks.
func resolveCredentials(cmd *cobra.Command) (string, string, io.Writer, error) {
	token, err := cmd.Flags().GetString(FlagToken)
	if err != nil {
		return "", "", nil, err
	}
	if token == "" {
		var storedErr error
		token, storedErr = authentication.LoadUsableStoredToken()
		if storedErr != nil {
			if errors.Is(storedErr, authentication.ErrTokenExpired) {
				return "", "", nil, storedErr
			}
			// A non-expiry read problem falls back to the environment, matching
			// the previous behavior for an unreadable or missing token file.
			token = ""
		}
	}
	if token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return "", "", nil, fmt.Errorf("a token is required: pass --%s, run \"jakapil-cli authenticate\", or set %s", FlagToken, EnvToken)
	}

	identityEndpoint, err := cmd.Flags().GetString(FlagEndpoint)
	if err != nil {
		return "", "", nil, err
	}
	// The flag carries a non-empty default value (DefaultEndpoint), yet it must
	// only win over the environment when explicitly set by the user. Otherwise
	// $OS_AUTH_URL would never be honored.
	if identityEndpoint == "" || !cmd.Flags().Changed(FlagEndpoint) {
		identityEndpoint = os.Getenv(EnvEndpoint)
	}
	if identityEndpoint == "" {
		identityEndpoint = DefaultEndpoint
	}

	var debugOut io.Writer
	if debug, _ := cmd.Flags().GetBool(FlagDebug); debug {
		debugOut = cmd.ErrOrStderr()
	}

	return token, identityEndpoint, debugOut, nil
}

// discoverCRaasEndpoint resolves the public CRaaS API endpoint for the requested
// service type from the Keystone service catalog. CRaaS is not regional, so an
// empty region is passed, matching any catalog entry.
func discoverCRaasEndpoint(identityEndpoint, token, serviceType string, debug io.Writer) (string, error) {
	return keystone.DiscoverEndpoint(keystone.Options{
		IdentityEndpoint: identityEndpoint,
		Token:            token,
		ServiceType:      serviceType,
		Debug:            debug,
	})
}
