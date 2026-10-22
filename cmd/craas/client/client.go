/*
Copyright © 2026 Kot1gruN
*/
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	craasclient "github.com/selectel/craas-go/pkg/v1/client"
	craasclientv2 "github.com/selectel/craas-go/pkg/v2/client"
	"github.com/spf13/cobra"
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
	EnvEndpoint = "SLT_AUTH_URL"

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

	// discoveryTimeout bounds the HTTP calls made during service discovery.
	discoveryTimeout = 30 * time.Second
)

// NewClient builds an authenticated craas-go ServiceClient for the V1 API from
// the given command. Instead of hitting a hard-coded CRaaS URL, it authenticates
// against the Keystone Identity v3 endpoint (--auth-url or SLT_AUTH_URL) with
// the provided token and resolves the CRaaS V1 API endpoint (service type
// "container-registry") from the returned service catalog.
// The token is taken from the "--token" flag or the SLT_KS_TOKEN environment
// variable; the identity endpoint from the "--auth-url" flag, otherwise from
// the SLT_AUTH_URL environment variable, or DefaultEndpoint.
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
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return "", "", nil, fmt.Errorf("a token is required: pass --%s or set %s", FlagToken, EnvToken)
	}

	identityEndpoint, err := cmd.Flags().GetString(FlagEndpoint)
	if err != nil {
		return "", "", nil, err
	}
	// The flag carries a non-empty default value (DefaultEndpoint), yet it must
	// only win over the environment when explicitly set by the user. Otherwise
	// SLT_AUTH_URL would never be honored.
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

// discoverCRaasEndpoint authenticates against the Keystone Identity v3 endpoint
// with the given token, extracts the service catalog and returns the public CRaaS
// API endpoint for the requested service type ("container-registry" for V1 or
// "container-registry-v2" for V2). CRaaS is not regional, so the single catalog
// entry is used regardless of its region field. When debug is non-nil the
// discovered catalog and CRaaS endpoints are printed to it.
func discoverCRaasEndpoint(identityEndpoint, token, serviceType string, debug io.Writer) (string, error) {
	provider, err := openstack.NewClient(identityEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid identity endpoint %q: %w", identityEndpoint, err)
	}
	provider.HTTPClient = http.Client{Timeout: discoveryTimeout}

	authOpts := &gophercloud.AuthOptions{
		IdentityEndpoint: identityEndpoint,
		TokenID:          token,
	}

	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	// With only a token and no scope, AuthenticateV3 performs an identity v3
	// token lookup (catalog + tokens) and wires up provider.EndpointLocator to
	// resolve endpoints from the discovered catalog.
	if err := openstack.AuthenticateV3(ctx, provider, authOpts, gophercloud.EndpointOpts{}); err != nil {
		return "", fmt.Errorf("service discovery via identity %q failed: %w", identityEndpoint, err)
	}

	crEndpoint, err := provider.EndpointLocator(gophercloud.EndpointOpts{
		Type:         serviceType,
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return "", fmt.Errorf("CRaaS endpoint for service type %q not found in service catalog: %w", serviceType, err)
	}

	// gophercloud.NormalizeURL appends a trailing slash to endpoints, but
	// selectel/craas-go clients require the endpoint to end in "/v1" (or "/v2").
	// A trailing slash would break the version-suffix check, so trim it.
	crEndpoint = strings.TrimRight(crEndpoint, "/")

	if debug != nil {
		printCRaasDiscoveryDebug(debug, provider, serviceType, crEndpoint)
	}

	return crEndpoint, nil
}

// printCRaasDiscoveryDebug dumps the raw service catalog and the CRaaS endpoints
// selected from it to w. It is used only for troubleshooting.
func printCRaasDiscoveryDebug(w io.Writer, provider *gophercloud.ProviderClient, serviceType, crEndpoint string) {
	catalog, err := extractCatalog(provider)
	if err != nil {
		fmt.Fprintf(w, "debug: cannot extract service catalog: %v\n", err)
		return
	}

	if raw, err := json.MarshalIndent(catalog, "", "  "); err == nil {
		fmt.Fprintf(w, "debug: identity service catalog:\n%s\n", raw)
	}

	fmt.Fprintln(w, "debug: CRaaS endpoints by catalog entry:")
	for _, entry := range catalog.Entries {
		if entry.Type != serviceType {
			continue
		}
		fmt.Fprintf(w, "  service %q (name=%s, id=%s):\n", entry.Type, entry.Name, entry.ID)
		for _, ep := range entry.Endpoints {
			fmt.Fprintf(w, "    interface=%-8s region=%s url=%s\n", ep.Interface, ep.Region, ep.URL)
		}
	}

	fmt.Fprintf(w, "debug: selected CRaaS endpoint (service=%q): %s\n", serviceType, crEndpoint)
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
