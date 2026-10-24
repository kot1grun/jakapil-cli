/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"errors"
	"fmt"
	"io"
	"os"

	v1 "github.com/selectel/mks-go/pkg/v1"
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

	// FlagRegion is the persistent flag name that selects the target region for
	// service discovery.
	FlagRegion = "region"

	// FlagDebug is the persistent flag name that enables debug output of the
	// discovered service catalog and MKS endpoints.
	FlagDebug = "debug"

	// EnvToken is the environment variable used as a fallback for FlagToken.
	EnvToken = "SLT_KS_TOKEN"

	// EnvEndpoint is the environment variable used as a fallback for FlagEndpoint.
	EnvEndpoint = "SLT_AUTH_URL"

	// EnvRegion is the environment variable used as a fallback for FlagRegion.
	EnvRegion = "SLT_REGION"

	// DefaultEndpoint is the Selectel Keystone Identity v3 endpoint used when
	// neither the flag nor the environment provides one. The Managed Kubernetes
	// API endpoint is then discovered from the service catalog.
	DefaultEndpoint = "https://cloud.api.selcloud.ru/identity/v3"

	// mksServiceType is the MKS service type as it appears in the Selectel
	// service catalog.
	mksServiceType = "managed-kubernetes"
)

// NewClient builds a MKS V1 ServiceClient from the given command. Instead of
// hitting a hard-coded MKS URL, it authenticates against the Keystone Identity
// v3 endpoint (--auth-url or SLT_AUTH_URL) with the provided token and resolves
// the MKS API endpoint from the returned service catalog, selecting the entry
// for the --region (or SLT_REGION) when set.
//
// The token is taken from the "--token" flag or the SLT_KS_TOKEN environment
// variable; the identity endpoint from the "--auth-url" flag, otherwise from
// the SLT_AUTH_URL environment variable, or DefaultEndpoint; the region from
// the "--region" flag, otherwise from the SLT_REGION environment variable.
func NewClient(cmd *cobra.Command) (*v1.ServiceClient, error) {
	token, err := cmd.Flags().GetString(FlagToken)
	if err == nil && token == "" {
		var storedErr error
		token, storedErr = authentication.LoadUsableStoredToken()
		if storedErr != nil {
			if errors.Is(storedErr, authentication.ErrTokenExpired) {
				return nil, storedErr
			}
			// A non-expiry read problem falls back to the environment, matching
			// the previous behavior for an unreadable or missing token file.
			token = ""
		}
	}
	if err != nil || token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return nil, fmt.Errorf("authentication token is required: pass --%s flag, run \"jakapil-cli authenticate\", or set %s env var", FlagToken, EnvToken)
	}

	identityEndpoint, err := cmd.Flags().GetString(FlagEndpoint)
	// The flag carries a non-empty default value (DefaultEndpoint), yet it must
	// only win over the environment when explicitly set by the user. Otherwise
	// SLT_AUTH_URL would never be honored.
	if identityEndpoint == "" || !cmd.Flags().Changed(FlagEndpoint) {
		identityEndpoint = os.Getenv(EnvEndpoint)
	}
	if identityEndpoint == "" {
		identityEndpoint = DefaultEndpoint
	}

	region, err := cmd.Flags().GetString(FlagRegion)
	if err != nil || region == "" {
		region = os.Getenv(EnvRegion)
	}

	var debugOut io.Writer
	if debug, _ := cmd.Flags().GetBool(FlagDebug); debug {
		debugOut = cmd.ErrOrStderr()
	}

	mksEndpoint, err := discoverMKSEndpoint(identityEndpoint, token, region, debugOut)
	if err != nil {
		return nil, err
	}

	return v1.NewMKSClientV1(token, mksEndpoint), nil
}

// discoverMKSEndpoint resolves the public MKS API endpoint for the requested
// region from the Keystone service catalog.
func discoverMKSEndpoint(identityEndpoint, token, region string, debug io.Writer) (string, error) {
	return keystone.DiscoverEndpoint(keystone.Options{
		IdentityEndpoint: identityEndpoint,
		Token:            token,
		ServiceType:      mksServiceType,
		Region:           region,
		Debug:            debug,
	})
}
