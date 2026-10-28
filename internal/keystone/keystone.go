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

	serviceEndpoint, err := provider.EndpointLocator(gophercloud.EndpointOpts{
		Type:         o.ServiceType,
		Region:       o.Region,
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return "", fmt.Errorf("endpoint for service type %q (region %q) not found in service catalog: %w",
			o.ServiceType, o.Region, err)
	}

	// gophercloud.NormalizeURL appends a trailing slash to endpoints, but the
	// Selectel clients (mks-go, craas-go) either validate a version suffix or
	// build URLs with plain concatenation; a trailing slash would break both.
	serviceEndpoint = strings.TrimRight(serviceEndpoint, "/")

	if o.Debug != nil {
		printDebug(o.Debug, provider, o, serviceEndpoint)
	}

	return serviceEndpoint, nil
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
