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
	v1 "github.com/selectel/mks-go/pkg/v1"
	"github.com/spf13/cobra"
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

	// discoveryTimeout bounds the HTTP calls made during service discovery.
	discoveryTimeout = 30 * time.Second
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
	if err != nil || token == "" {
		token = os.Getenv(EnvToken)
	}
	if token == "" {
		return nil, fmt.Errorf("authentication token is required: pass --%s flag or set %s env var", FlagToken, EnvToken)
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

// discoverMKSEndpoint authenticates against the Keystone Identity v3 endpoint
// with the given token, extracts the service catalog and returns the public MKS
// API endpoint for the requested region (or the first available one when region
// is empty). When debug is non-nil the discovered catalog and MKS endpoints are
// printed to it.
func discoverMKSEndpoint(identityEndpoint, token, region string, debug io.Writer) (string, error) {
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

	mksEndpoint, err := provider.EndpointLocator(gophercloud.EndpointOpts{
		Type:         mksServiceType,
		Region:       region,
		Availability: gophercloud.AvailabilityPublic,
	})
	if err != nil {
		return "", fmt.Errorf("MKS endpoint for region %q not found in service catalog: %w", region, err)
	}

	// gophercloud.NormalizeURL appends a trailing slash to endpoints, but
	// selectel/mks-go builds request URLs as strings.Join([]string{Endpoint,
	// "clusters", id}, "/"). A trailing slash would produce a double slash
	// (e.g. ".../v1//clusters/{id}") and a 404. Trim it before use.
	mksEndpoint = strings.TrimRight(mksEndpoint, "/")

	if debug != nil {
		printDiscoveryDebug(debug, provider, region, mksEndpoint)
	}

	return mksEndpoint, nil
}

// printDiscoveryDebug dumps the raw service catalog and the MKS endpoints
// selected from it to w. It is used only for troubleshooting.
func printDiscoveryDebug(w io.Writer, provider *gophercloud.ProviderClient, region, mksEndpoint string) {
	catalog, err := extractCatalog(provider)
	if err != nil {
		fmt.Fprintf(w, "debug: cannot extract service catalog: %v\n", err)
		return
	}

	if raw, err := json.MarshalIndent(catalog, "", "  "); err == nil {
		fmt.Fprintf(w, "debug: identity %q service catalog:\n%s\n", region, raw)
	}

	fmt.Fprintln(w, "debug: MKS endpoints by catalog entry:")
	for _, entry := range catalog.Entries {
		if entry.Type != mksServiceType {
			continue
		}
		fmt.Fprintf(w, "  service %q (name=%s, id=%s):\n", entry.Type, entry.Name, entry.ID)
		for _, ep := range entry.Endpoints {
			fmt.Fprintf(w, "    interface=%-8s region=%s url=%s\n", ep.Interface, ep.Region, ep.URL)
		}
	}

	fmt.Fprintf(w, "debug: selected MKS endpoint (region=%q): %s\n", region, mksEndpoint)
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
