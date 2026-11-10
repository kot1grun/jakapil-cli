/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/selectel/go-selvpcclient/v5/selvpcclient"
	"github.com/selectel/go-selvpcclient/v5/selvpcclient/clients"
	clientservices "github.com/selectel/go-selvpcclient/v5/selvpcclient/clients/services"
	"github.com/spf13/cobra"

	"jakapil-cli/internal/authentication"
)

const (
	// FlagToken is the persistent flag name that holds the Selectel Keystone API token.
	// The Cloud Platform Projects API (Resell v2) requires an IAM account-scoped token,
	// obtained with "jakapil-cli authenticate token issue --account-scope" and normally
	// persisted by that command.
	FlagToken = "token"

	// FlagEndpoint is the persistent flag name that holds the Selectel Keystone
	// Identity v3 auth URL used to resolve the Resell service endpoint from the
	// service catalog.
	FlagEndpoint = "auth-url"

	// FlagDebug is the persistent flag name that enables debug output of the
	// discovered Resell endpoint.
	FlagDebug = "debug"

	// EnvToken is the environment variable used as a fallback for FlagToken.
	EnvToken = "SLT_KS_TOKEN"

	// EnvEndpoint is the environment variable used as a fallback for FlagEndpoint.
	// It follows the openrc/Keystone convention (OS_AUTH_URL) so that a sourced
	// openrc file is reused.
	EnvEndpoint = "OS_AUTH_URL"

	// DefaultEndpoint is the Selectel Keystone Identity v3 endpoint used when
	// neither the flag nor the environment provides one.
	DefaultEndpoint = "https://cloud.api.selcloud.ru/identity/v3"

	// resellServiceType is the Resell service type as it appears in the Selectel
	// service catalog. The Cloud Platform Projects API lives under /vpc/resell.
	resellServiceType = "resell"

	// RegionFlag is shared by the persistent flags so that the selectel Projects
	// command, which is not regional by nature, can still pick the Resell catalog
	// entry (some accounts advertise several regions). It mirrors the name used by
	// the VPC/MKS command groups.
	RegionFlag = "region"
)

// NewClient builds a go-selvpcclient Client for the Cloud Platform Projects API
// (Resell v2) from the given command.
//
// Unlike selvpcclient.NewClient, which authenticates with a username and
// password, this builder injects an already-issued Keystone token (taken from
// the "--token" flag, otherwise from the persisted stored token or the
// SLT_KS_TOKEN environment variable). The token must be account-scoped, i.e.
// issued with "jakapil-cli authenticate token issue --account-scope": that is
// the only scope the Projects endpoints accept.
//
// The Resell service endpoint is resolved from the Keystone service catalog
// (auth URL from the "--auth-url" flag, otherwise $OS_AUTH_URL, otherwise
// DefaultEndpoint), mirroring how the other Selectel SDK clients in this repo
// discover their endpoints.
func NewClient(cmd *cobra.Command) (*selvpcclient.Client, error) {
	token, identityEndpoint, region, err := resolveCredentials(cmd)
	if err != nil {
		return nil, err
	}

	serviceClient := &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{TokenID: token},
		Endpoint:       normalizeEndpoint(identityEndpoint),
	}

	catalogService, err := clientservices.NewCatalogService(serviceClient)
	if err != nil {
		return nil, err
	}

	// If the caller did not pin a region, pick the first public Resell endpoint
	// advertised in the catalog so region matching never fails spuriously.
	if region == "" {
		region = resolveCatalogRegion(catalogService)
		if region == "" {
			return nil, fmt.Errorf("no public endpoint for service type %q found in service catalog", resellServiceType)
		}
	}

	if debug, _ := cmd.Flags().GetBool(FlagDebug); debug {
		debugEndpoint(cmd.ErrOrStderr(), catalogService, region)
	}

	requestService := clientservices.NewRequestService(serviceClient)

	return &selvpcclient.Client{
		Resell: clients.NewResellClient(requestService, catalogService, region),
	}, nil
}

// resolveCredentials returns the token, identity endpoint and (optional) target
// region resolved from the command flags with the environment variables as
// fallbacks.
func resolveCredentials(cmd *cobra.Command) (token, identityEndpoint, region string, err error) {
	token, err = cmd.Flags().GetString(FlagToken)
	if err == nil && token == "" {
		var storedErr error
		// The Cloud Platform Projects API requires an account-scoped token
		// (issued with "authenticate token issue --account-scope"), which is not
		// what LoadUsableStoredToken returns: that helper yields the project
		// token consumed by MKS/CRaaS. Use the account-scoped loader instead.
		token, storedErr = authentication.LoadUsableAccountToken()
		if storedErr != nil {
			if errors.Is(storedErr, authentication.ErrTokenExpired) {
				return "", "", "", storedErr
			}
			// A non-expiry read problem falls back to the environment.
			token = ""
		}
	}
	if err != nil || token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return "", "", "", fmt.Errorf("account-scoped authentication token is required: pass --%s, run \"jakapil-cli authenticate token issue --account-scope\", or set %s env var", FlagToken, EnvToken)
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

	region, _ = cmd.Flags().GetString(RegionFlag)
	return token, identityEndpoint, region, nil
}

// resolveCatalogRegion scans the already-fetched catalog for the first public
// endpoint of the Resell service and returns its region name. An empty string is
// returned when none is advertised (for example a non-regional catalog entry).
func resolveCatalogRegion(catalogService *clientservices.CatalogService) string {
	catalog, err := catalogService.GetCatalog()
	if err != nil {
		return ""
	}
	for _, entry := range catalog.Entries {
		if entry.Type != resellServiceType {
			continue
		}
		for _, ep := range entry.Endpoints {
			if ep.Interface == string(gophercloud.AvailabilityPublic) {
				return ep.RegionID
			}
		}
	}
	return ""
}

// debugEndpoint prints the Resell endpoint selected by the catalog service for
// the given region, mirroring the --debug output of the other command groups.
func debugEndpoint(w io.Writer, catalogService *clientservices.CatalogService, region string) {
	endpoint, err := catalogService.GetEndpoint(resellServiceType, region)
	if err != nil {
		fmt.Fprintf(w, "debug: resell endpoint lookup failed: %v\n", err)
		return
	}
	fmt.Fprintf(w, "debug: selected endpoint (service=%q, region=%q): %s\n", resellServiceType, region, endpoint.URL)
}

// normalizeEndpoint appends a trailing slash so go-selvpcclient/gophercloud can
// resolve relative request URLs against it (gophercloud service URLs must end
// with a slash).
func normalizeEndpoint(endpoint string) string {
	if strings.HasSuffix(endpoint, "/") {
		return endpoint
	}
	return endpoint + "/"
}
