/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
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

	// FlagAllRegions is the flag name that selects every region the
	// service advertises in the catalog instead of a single --region. It is
	// registered as a local flag on the list subcommands only.
	FlagAllRegions = "all-regions"

	// FlagDebug is the persistent flag name that enables debug output of the
	// discovered VPC/network endpoint.
	FlagDebug = "debug"

	// EnvToken is the environment variable used as a fallback for FlagToken.
	EnvToken = "SLT_KS_TOKEN"

	// EnvEndpoint is the environment variable used as a fallback for FlagEndpoint.
	// It follows the openrc/Keystone convention (OS_AUTH_URL) so that a sourced
	// openrc file is reused.
	EnvEndpoint = "OS_AUTH_URL"

	// EnvRegion is the environment variable used as a fallback for FlagRegion.
	// It follows the openrc/Keystone convention (OS_REGION_NAME).
	EnvRegion = "OS_REGION_NAME"

	// DefaultEndpoint is the Selectel Keystone Identity v3 endpoint used when
	// neither the flag nor the environment provides one.
	DefaultEndpoint = "https://cloud.api.selcloud.ru/identity/v3"

	// networkServiceType is the Gophercloud networking service type as it
	// appears in the service catalog.
	networkServiceType = "network"
)

// NewNetworkServiceClient builds a gophercloud networking v2 ServiceClient for
// the VPC service from the given command. Instead of hitting a hard-coded URL,
// it authenticates against the Keystone Identity v3 endpoint (--auth-url or
// $OS_AUTH_URL) with the provided token and resolves the public networking
// ("network") endpoint from the returned service catalog.
//
// The token is taken from the "--token" flag, otherwise from the persisted
// stored token or the SLT_KS_TOKEN environment variable; the identity endpoint
// from the "--auth-url" flag, otherwise from $OS_AUTH_URL or DefaultEndpoint;
// the region from the "--region" flag, otherwise from $OS_REGION_NAME.
func NewNetworkServiceClient(cmd *cobra.Command) (*gophercloud.ServiceClient, error) {
	token, identityEndpoint, debug, err := networkCredentials(cmd)
	if err != nil {
		return nil, err
	}

	provider, err := keystone.Authenticate(keystone.Options{
		IdentityEndpoint: identityEndpoint,
		Token:            token,
		Debug:            debug,
	})
	if err != nil {
		return nil, err
	}

	return openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Region:       resolveRegion(cmd),
		Availability: gophercloud.AvailabilityPublic,
	})
}

// NewNetworkServiceClients authenticates once and returns a networking v2
// ServiceClient for every public region the "network" service advertises in the
// catalog, together with that region list sorted by name. It backs the
// "--all-regions" list commands.
func NewNetworkServiceClients(cmd *cobra.Command) ([]string, map[string]*gophercloud.ServiceClient, error) {
	token, identityEndpoint, debug, err := networkCredentials(cmd)
	if err != nil {
		return nil, nil, err
	}

	provider, err := keystone.Authenticate(keystone.Options{
		IdentityEndpoint: identityEndpoint,
		Token:            token,
		Debug:            debug,
	})
	if err != nil {
		return nil, nil, err
	}

	regions, err := keystone.RegionsFromProvider(provider, networkServiceType)
	if err != nil {
		return nil, nil, err
	}

	clients := make(map[string]*gophercloud.ServiceClient, len(regions))
	for _, region := range regions {
		sc, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
			Region:       region,
			Availability: gophercloud.AvailabilityPublic,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("build network client for region %q: %w", region, err)
		}
		clients[region] = sc
	}
	return regions, clients, nil
}

// networkCredentials resolves the token, identity endpoint and debug writer
// shared by every VPC client builder.
func networkCredentials(cmd *cobra.Command) (token, identityEndpoint string, debug io.Writer, err error) {
	token, err = resolveToken(cmd)
	if err != nil {
		return "", "", nil, err
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

	if debugOn, _ := cmd.Flags().GetBool(FlagDebug); debugOn {
		debug = cmd.ErrOrStderr()
	}
	return token, identityEndpoint, debug, nil
}

// EffectiveRegion returns the region a single-region list actually used: the
// explicitly selected --region/$OS_REGION_NAME when set, otherwise the region
// gophercloud resolved from the service catalog (the first public endpoint). It
// is used to name the region in "No resources found in region <region>" notices
// even when the user relied on catalog defaulting. When the client does not
// carry a catalog (plain test clients), it degrades to the generic notice.
func EffectiveRegion(cmd *cobra.Command, c *gophercloud.ServiceClient) string {
	if region := resolveRegion(cmd); region != "" {
		return region
	}
	if c == nil || c.ProviderClient == nil {
		return ""
	}
	region, err := keystone.RegionMatchingEndpoint(c.ProviderClient, c.Endpoint)
	if err != nil {
		return ""
	}
	return region
}

// resolveRegion returns the target region from the "--region" flag, otherwise
// from the $OS_REGION_NAME environment variable.
func resolveRegion(cmd *cobra.Command) string {
	region, _ := cmd.Flags().GetString(FlagRegion)
	if region == "" {
		region = os.Getenv(EnvRegion)
	}
	return region
}

// resolveToken returns the API token from the "--token" flag, falling back to
// the persisted stored token (when it has not expired) and then to the
// SLT_KS_TOKEN environment variable.
func resolveToken(cmd *cobra.Command) (string, error) {
	token, err := cmd.Flags().GetString(FlagToken)
	if err == nil && token == "" {
		var storedErr error
		token, storedErr = authentication.LoadUsableStoredToken()
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
		return "", fmt.Errorf("authentication token is required: pass --%s flag, run \"jakapil-cli authenticate token issue\", or set %s env var", FlagToken, EnvToken)
	}
	return token, nil
}
