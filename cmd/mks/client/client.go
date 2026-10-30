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

	// FlagAllRegions is the persistent flag name that selects every region the
	// service advertises in the catalog instead of a single --region.
	FlagAllRegions = "all-regions"

	// FlagDebug is the persistent flag name that enables debug output of the
	// discovered service catalog and MKS endpoints.
	FlagDebug = "debug"

	// EnvToken is the environment variable used as a fallback for FlagToken.
	EnvToken = "SLT_KS_TOKEN"

	// EnvEndpoint is the environment variable used as a fallback for FlagEndpoint.
	// It follows the openrc/Keystone convention (OS_AUTH_URL) so that a sourced
	// openrc file is reused instead of a Selectel-specific name.
	EnvEndpoint = "OS_AUTH_URL"

	// EnvRegion is the environment variable used as a fallback for FlagRegion.
	// It follows the openrc/Keystone convention (OS_REGION_NAME) instead of a
	// Selectel-specific name so that a sourced openrc file reuses the region.
	EnvRegion = "OS_REGION_NAME"

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
// v3 endpoint (--auth-url or $OS_AUTH_URL) with the provided token and resolves
// the MKS API endpoint from the returned service catalog, selecting the entry
// for the --region (or $OS_REGION_NAME) when set.
//
// The token is taken from the "--token" flag or the SLT_KS_TOKEN environment
// variable; the identity endpoint from the "--auth-url" flag, otherwise from
// the $OS_AUTH_URL environment variable, or DefaultEndpoint; the region from
// the "--region" flag, otherwise from the $OS_REGION_NAME environment variable.
func NewClient(cmd *cobra.Command) (*v1.ServiceClient, error) {
	token, identityEndpoint, debugOut, err := mksCredentials(cmd)
	if err != nil {
		return nil, err
	}

	mksEndpoint, err := discoverMKSEndpoint(identityEndpoint, token, resolveMksRegion(cmd), debugOut)
	if err != nil {
		return nil, err
	}

	return v1.NewMKSClientV1(token, mksEndpoint), nil
}

// NewClients authenticates once and returns a MKS V1 ServiceClient for every
// public region the "managed-kubernetes" service advertises in the catalog,
// together with that region list sorted by name. It backs the "--all-regions"
// list commands.
func NewClients(cmd *cobra.Command) ([]string, map[string]*v1.ServiceClient, error) {
	token, identityEndpoint, debugOut, err := mksCredentials(cmd)
	if err != nil {
		return nil, nil, err
	}

	provider, err := keystone.Authenticate(keystone.Options{
		IdentityEndpoint: identityEndpoint,
		Token:            token,
		Debug:            debugOut,
	})
	if err != nil {
		return nil, nil, err
	}

	regions, err := keystone.RegionsFromProvider(provider, mksServiceType)
	if err != nil {
		return nil, nil, err
	}

	clients := make(map[string]*v1.ServiceClient, len(regions))
	for _, region := range regions {
		mksEndpoint, err := keystone.ResolveEndpointFromProvider(provider, mksServiceType, region)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve MKS endpoint for region %q: %w", region, err)
		}
		clients[region] = v1.NewMKSClientV1(token, mksEndpoint)
	}
	return regions, clients, nil
}

// mksCredentials resolves the token, identity endpoint and debug writer shared
// by every MKS client builder.
func mksCredentials(cmd *cobra.Command) (token, identityEndpoint string, debugOut io.Writer, err error) {
	token, err = cmd.Flags().GetString(FlagToken)
	if err == nil && token == "" {
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
	if err != nil || token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return "", "", nil, fmt.Errorf("authentication token is required: pass --%s flag, run \"jakapil-cli authenticate token issue\", or set %s env var", FlagToken, EnvToken)
	}

	identityEndpoint, _ = cmd.Flags().GetString(FlagEndpoint)
	// The flag carries a non-empty default value (DefaultEndpoint), yet it must
	// only win over the environment when explicitly set by the user. Otherwise
	// $OS_AUTH_URL would never be honored.
	if identityEndpoint == "" || !cmd.Flags().Changed(FlagEndpoint) {
		identityEndpoint = os.Getenv(EnvEndpoint)
	}
	if identityEndpoint == "" {
		identityEndpoint = DefaultEndpoint
	}

	if debug, _ := cmd.Flags().GetBool(FlagDebug); debug {
		debugOut = cmd.ErrOrStderr()
	}
	return token, identityEndpoint, debugOut, nil
}

// resolveMksRegion returns the target region from the "--region" flag,
// otherwise from the $OS_REGION_NAME environment variable.
func resolveMksRegion(cmd *cobra.Command) string {
	region, err := cmd.Flags().GetString(FlagRegion)
	if err != nil || region == "" {
		region = os.Getenv(EnvRegion)
	}
	return region
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
