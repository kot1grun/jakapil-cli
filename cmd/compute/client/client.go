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
	// discovered compute endpoint.
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

	// computeServiceType is the Gophercloud compute service type as it
	// appears in the service catalog.
	computeServiceType = "compute"

	// EnvComputeAPIVersion is the environment variable that overrides the
	// Nova microversion sent on every compute request. If unset, the client
	// pins the maximum microversion supported by the selected region
	// ("latest"), which keeps behavior aligned with each region's capabilities
	// and is safe because Nova microversions are cumulative (new versions are
	// only added, never removed). Pinning an explicit value (e.g. "2.93") is
	// useful for reproducibility or when "latest" rolls in a breaking schema.
	//
	// Note: Selectel exposes high-frequency flavor options (highfreq,
	// dedicated_cpus, vcpus_vendor) only at a recent microversion; older
	// schemas reject them as unexpected properties.
	EnvComputeAPIVersion = "JAKAPIL_COMPUTE_API_VERSION"
)

// resolveComputeAPIVersion returns the Nova microversion to advertise on
// compute requests. It honors JAKAPIL_COMPUTE_API_VERSION when set, otherwise
// defers to the server's maximum supported microversion ("latest").
func resolveComputeAPIVersion() string {
	if v := os.Getenv(EnvComputeAPIVersion); v != "" {
		return v
	}
	return "latest"
}

// NewComputeServiceClient builds a gophercloud compute v2 ServiceClient for the
// Compute service from the given command. Instead of hitting a hard-coded URL,
// it authenticates against the Keystone Identity v3 endpoint (--auth-url or
// $OS_AUTH_URL) with the provided token and resolves the public compute endpoint
// from the returned service catalog.
//
// The token is taken from the "--token" flag, otherwise from the persisted
// stored token or the SLT_KS_TOKEN environment variable; the identity endpoint
// from the "--auth-url" flag, otherwise from $OS_AUTH_URL or DefaultEndpoint;
// the region from the "--region" flag, otherwise from $OS_REGION_NAME.
func NewComputeServiceClient(cmd *cobra.Command) (*gophercloud.ServiceClient, error) {
	token, identityEndpoint, debug, err := computeCredentials(cmd)
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

	sc, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region:       resolveRegion(cmd),
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return nil, err
	}
	sc.Microversion = resolveComputeAPIVersion()
	return sc, nil
}

// NewComputeServiceClients authenticates once and returns a compute v2
// ServiceClient for every public region the "compute" service advertises in the
// catalog, together with that region list sorted by name. It backs the
// "--all-regions" list commands.
func NewComputeServiceClients(cmd *cobra.Command) ([]string, map[string]*gophercloud.ServiceClient, error) {
	token, identityEndpoint, debug, err := computeCredentials(cmd)
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

	regions, err := keystone.RegionsFromProvider(provider, computeServiceType)
	if err != nil {
		return nil, nil, err
	}

	clients := make(map[string]*gophercloud.ServiceClient, len(regions))
	for _, region := range regions {
		sc, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
			Region:       region,
			Availability: gophercloud.AvailabilityPublic,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("build compute client for region %q: %w", region, err)
		}
		sc.Microversion = resolveComputeAPIVersion()
		clients[region] = sc
	}
	return regions, clients, nil
}

// computeCredentials resolves the token, identity endpoint and debug writer
// shared by every compute client builder.
func computeCredentials(cmd *cobra.Command) (token, identityEndpoint string, debug io.Writer, err error) {
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
