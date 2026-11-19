/*
Copyright © 2026 Kot1gruN
*/
package publicsubnet

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/selectel/go-selvpcclient/v5/selvpcclient"
	"github.com/selectel/go-selvpcclient/v5/selvpcclient/clients"
	clientservices "github.com/selectel/go-selvpcclient/v5/selvpcclient/clients/services"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
	"jakapil-cli/internal/authentication"
	"jakapil-cli/internal/keystone"
)

// resellServiceType is the Resell service type as it appears in the Selectel
// service catalog. Public subnets live under the Projects (Resell v2) API, the
// same service that backs "selproject".
const resellServiceType = "resell"

// NewResellClient builds a go-selvpcclient Client for the Public Subnets API
// (Resell v2) from the given command. It reuses the persistent "vpc" flags
// (--token/--auth-url/--region/--debug). The Subnets API
// (https://docs.selectel.ru/api/cloud-projects-and-resources/#tag/Subnets)
// accepts a project-scoped token, so the project-scoped stored token loader is
// used (the same scope the MKS/CRaaS commands consume).
//
// The Resell endpoint is resolved from the Keystone service catalog scoped to
// the --region flag, or to the first public Resell endpoint when the flag is
// unset.
func NewResellClient(cmd *cobra.Command) (*selvpcclient.Client, error) {
	token, err := resolveProjectToken(cmd)
	if err != nil {
		return nil, err
	}

	identityEndpoint := resolveEndpoint(cmd)
	region, _ := cmd.Flags().GetString(client.FlagRegion)

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

	if debug, _ := cmd.Flags().GetBool(client.FlagDebug); debug {
		debugEndpoint(cmd.ErrOrStderr(), catalogService, region)
	}

	requestService := clientservices.NewRequestService(serviceClient)

	return &selvpcclient.Client{
		Resell: clients.NewResellClient(requestService, catalogService, region),
	}, nil
}

// resolveProjectToken returns the API token from the "--token" flag, falling
// back to the persisted project-scoped token (when it has not expired) and then
// to the SLT_KS_TOKEN environment variable. The Public Subnets API lives on the
// Resell service but accepts a project-scoped token, so the project-scoped
// loader (LoadUsableStoredToken) is used.
func resolveProjectToken(cmd *cobra.Command) (string, error) {
	token, err := cmd.Flags().GetString(client.FlagToken)
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
		token = os.Getenv(client.EnvToken)
	}
	if token == "" {
		return "", fmt.Errorf("authentication token is required: pass --%s, run \"jakapil-cli authenticate token issue\", or set %s env var", client.FlagToken, client.EnvToken)
	}
	return token, nil
}

// resolveProjectID returns the ID of the project a project-scoped token is
// scoped to, resolving it from the Keystone Identity service. It is used by the
// "create" command when the user did not pass a project ID explicitly: the
// Public Subnets API is project-scoped, so the token already carries the target
// project. An empty result with a nil error means the token is not scoped to a
// project (for example an account-scoped token).
func resolveProjectID(cmd *cobra.Command) (string, error) {
	token, err := resolveProjectToken(cmd)
	if err != nil {
		return "", err
	}

	provider, err := keystone.Authenticate(keystone.Options{
		IdentityEndpoint: resolveEndpoint(cmd),
		Token:            token,
	})
	if err != nil {
		return "", err
	}
	return keystone.ProjectIDFromProvider(provider)
}

// resolveEndpoint returns the identity endpoint from the --auth-url flag,
// falling back to $OS_AUTH_URL and then to the DefaultEndpoint. The flag
// carries a non-empty default, so it must only win over the environment when
// explicitly set by the user.
func resolveEndpoint(cmd *cobra.Command) string {
	endpoint, _ := cmd.Flags().GetString(client.FlagEndpoint)
	if endpoint == "" || !cmd.Flags().Changed(client.FlagEndpoint) {
		endpoint = os.Getenv(client.EnvEndpoint)
	}
	if endpoint == "" {
		endpoint = client.DefaultEndpoint
	}
	return endpoint
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
func debugEndpoint(w interface{ Write([]byte) (int, error) }, catalogService *clientservices.CatalogService, region string) {
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
