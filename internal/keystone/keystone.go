/*
Package keystone resolves Selectel service endpoints from the Keystone Identity
v3 service catalog. Both the MKS and CRaaS clients use it to avoid hard-coding
API URLs: a token lookup against the identity endpoint returns the catalog, from
which the public endpoint for the requested service type is selected.
*/
package keystone

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

// discoveryTimeout bounds the HTTP calls made during service discovery.
const discoveryTimeout = 30 * time.Second

// Options configures service endpoint discovery through the Keystone Identity
// v3 service catalog.
type Options struct {
	// IdentityEndpoint is the Keystone Identity v3 auth URL.
	IdentityEndpoint string
	// Token is the Selectel Keystone API token used to look up the catalog.
	Token string
	// ServiceType is the service type to resolve from the catalog, for example
	// "managed-kubernetes", "container-registry" or "container-registry-v2".
	ServiceType string
	// Region narrows endpoint selection. Empty matches any region (useful for
	// non-regional services such as CRaaS).
	Region string
	// Debug, when non-nil, receives the discovered service catalog and the
	// selected endpoint for troubleshooting.
	Debug io.Writer
}

// Authenticate authenticates against IdentityEndpoint with Token and returns the
// authenticated ProviderClient. Authentication wires up provider.EndpointLocator
// from the resulting service catalog, which lets gophercloud ServiceClients
// (for example openstack.NewNetworkV2 for the VPC networking service) resolve
// their endpoints without repeating the lookup.
func Authenticate(o Options) (*gophercloud.ProviderClient, error) {
	provider, err := openstack.NewClient(o.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid identity endpoint %q: %w", o.IdentityEndpoint, err)
	}
	provider.HTTPClient = http.Client{Timeout: discoveryTimeout}

	authOpts := &gophercloud.AuthOptions{
		IdentityEndpoint: o.IdentityEndpoint,
		TokenID:          o.Token,
	}

	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	// With only a token and no scope, AuthenticateV3 performs an identity v3
	// token lookup (catalog + tokens) and wires up provider.EndpointLocator to
	// resolve endpoints from the discovered catalog.
	if err := openstack.AuthenticateV3(ctx, provider, authOpts, gophercloud.EndpointOpts{}); err != nil {
		return nil, fmt.Errorf("authentication via identity %q failed: %w", o.IdentityEndpoint, err)
	}

	return provider, nil
}

// DiscoverEndpoint authenticates against IdentityEndpoint with Token, extracts
// the service catalog and returns the public endpoint URL for ServiceType,
// filtered by Region when it is non-empty. The returned URL has no trailing
// slash so version-suffixed endpoints (".../v1") that are validated by the
// Selectel API clients are not broken.
func DiscoverEndpoint(o Options) (string, error) {
	provider, err := Authenticate(o)
	if err != nil {
		return "", err
	}

	serviceEndpoint, err := ResolveEndpointFromProvider(provider, o.ServiceType, o.Region)
	if err != nil {
		return "", err
	}

	if o.Debug != nil {
		printDebug(o.Debug, provider, o, serviceEndpoint)
	}

	return serviceEndpoint, nil
}

// ResolveEndpointFromProvider returns the public endpoint URL for a service
// type and region from an already-authenticated provider, reusing the catalog
// fetched during authentication so no second lookup is needed. The returned URL
// has no trailing slash.
func ResolveEndpointFromProvider(provider *gophercloud.ProviderClient, serviceType, region string) (string, error) {
	serviceEndpoint, err := provider.EndpointLocator(gophercloud.EndpointOpts{
		Type:         serviceType,
		Region:       region,
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return "", fmt.Errorf("endpoint for service type %q (region %q) not found in service catalog: %w",
			serviceType, region, err)
	}

	return strings.TrimRight(serviceEndpoint, "/"), nil
}

// RegionsFromProvider returns the sorted, de-duplicated list of public regions
// advertised for serviceType in the already-authenticated provider's catalog.
// An empty region endpoint (non-regional services such as CRaaS) is ignored.
func RegionsFromProvider(provider *gophercloud.ProviderClient, serviceType string) ([]string, error) {
	catalog, err := extractCatalog(provider)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var regions []string
	for _, entry := range catalog.Entries {
		if entry.Type != serviceType {
			continue
		}
		for _, ep := range entry.Endpoints {
			if ep.Region == "" || ep.Interface != string(gophercloud.AvailabilityPublic) {
				continue
			}
			if !seen[ep.Region] {
				seen[ep.Region] = true
				regions = append(regions, ep.Region)
			}
		}
	}
	if len(regions) == 0 {
		return nil, fmt.Errorf("no public endpoints for service type %q found in service catalog", serviceType)
	}
	sort.Strings(regions)
	return regions, nil
}

// RegionMatchingEndpoint returns the region whose public endpoint URL in the
// service catalog matches endpoint (trailing slashes ignored). Endpoint URLs are
// unique across the catalog, so no service type is required. It is used to report
// which region a service client actually resolved to when none was requested
// explicitly (gophercloud then uses the first public endpoint from the catalog).
func RegionMatchingEndpoint(provider *gophercloud.ProviderClient, endpoint string) (string, error) {
	catalog, err := extractCatalog(provider)
	if err != nil {
		return "", err
	}
	target := strings.TrimRight(endpoint, "/")
	for _, entry := range catalog.Entries {
		for _, ep := range entry.Endpoints {
			if ep.Region == "" || ep.Interface != string(gophercloud.AvailabilityPublic) {
				continue
			}
			if strings.TrimRight(ep.URL, "/") == target {
				return ep.Region, nil
			}
		}
	}
	return "", fmt.Errorf("no public endpoint in the service catalog matches %q", endpoint)
}

// RegionMatchingEndpointAuthenticated authenticates with o and returns which
// region's public endpoint matches endpoint. It is the counterpart of
// RegionMatchingEndpoint for callers that do not retain the authenticated
// ProviderClient (for example the MKS SDK client, which keeps only the resolved
// endpoint URL).
func RegionMatchingEndpointAuthenticated(o Options, endpoint string) (string, error) {
	provider, err := Authenticate(o)
	if err != nil {
		return "", err
	}
	return RegionMatchingEndpoint(provider, endpoint)
}

// DiscoverRegions authenticates against IdentityEndpoint with Token and returns
// the sorted list of public regions advertised for o.ServiceType in the service
// catalog. It is the counterpart of DiscoverEndpoint for the --all-regions mode.
func DiscoverRegions(o Options) ([]string, error) {
	provider, err := Authenticate(o)
	if err != nil {
		return nil, err
	}
	return RegionsFromProvider(provider, o.ServiceType)
}

// printDebug dumps the raw service catalog and the endpoints selected from it
// to w. It is used only for troubleshooting.
func printDebug(w io.Writer, provider *gophercloud.ProviderClient, o Options, serviceEndpoint string) {
	catalog, err := extractCatalog(provider)
	if err != nil {
		fmt.Fprintf(w, "debug: cannot extract service catalog: %v\n", err)
		return
	}

	if raw, err := json.MarshalIndent(catalog, "", "  "); err == nil {
		fmt.Fprintf(w, "debug: identity service catalog:\n%s\n", raw)
	}

	fmt.Fprintln(w, "debug: endpoints by catalog entry:")
	for _, entry := range catalog.Entries {
		if entry.Type != o.ServiceType {
			continue
		}
		fmt.Fprintf(w, "  service %q (name=%s, id=%s):\n", entry.Type, entry.Name, entry.ID)
		for _, ep := range entry.Endpoints {
			fmt.Fprintf(w, "    interface=%-8s region=%s url=%s\n", ep.Interface, ep.Region, ep.URL)
		}
	}

	fmt.Fprintf(w, "debug: selected endpoint (service=%q, region=%q): %s\n", o.ServiceType, o.Region, serviceEndpoint)
}

// extractCatalog returns the service catalog produced during authentication.
func extractCatalog(provider *gophercloud.ProviderClient) (*tokens3.ServiceCatalog, error) {
	switch r := provider.GetAuthResult().(type) {
	case tokens3.CreateResult:
		return r.ExtractServiceCatalog()
	case tokens3.GetResult:
		return r.ExtractServiceCatalog()
	default:
		return nil, fmt.Errorf("unexpected auth result type %T", provider.GetAuthResult())
	}
}
