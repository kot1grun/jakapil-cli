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
	publicnetapi "github.com/selectel/public-net-api-go/pkg/v1"
	"github.com/spf13/cobra"

	"jakapil-cli/cmd/vpc/client"
	"jakapil-cli/internal/authentication"
	"jakapil-cli/internal/keystone"
)

// publicNetServiceType is the Public Net service type as it appears in the
// Selectel Keystone service catalog. Public Net is a regional service, so its
// endpoint is resolved from the catalog scoped to the --region flag, exactly
// like the other VPC objects.
const publicNetServiceType = "public-net-api"

// NewClient builds a Public Net API v1 client from the given command. It
// reuses the persistent "vpc" flags (--token/--auth-url/--region/--debug).
// The Public Net API accepts a project-scoped token (the public ports belong
// to a project), so the project-scoped stored token loader is used (the same
// scope the MKS/CRaaS/public subnet commands consume).
//
// The endpoint is resolved from the Keystone service catalog scoped to the
// --region flag, or to the first public Public Net endpoint when the flag is
// unset.
func NewClient(cmd *cobra.Command) (*publicnetapi.PublicNetAPIClient, error) {
	provider, token, err := authenticatedProvider(cmd)
	if err != nil {
		return nil, err
	}

	region, _ := cmd.Flags().GetString(client.FlagRegion)
	endpoint, err := keystone.ResolveEndpointFromProvider(provider, publicNetServiceType, region)
	if err != nil {
		return nil, err
	}

	return newAPIClient(endpoint, token)
}

// NewClients returns one Public Net client per public region the service
// advertises in the catalog, together with the sorted region list. It backs
// the --all-regions list mode.
func NewClients(cmd *cobra.Command) ([]string, []*publicnetapi.PublicNetAPIClient, error) {
	provider, token, err := authenticatedProvider(cmd)
	if err != nil {
		return nil, nil, err
	}

	regions, err := keystone.RegionsFromProvider(provider, publicNetServiceType)
	if err != nil {
		return nil, nil, err
	}

	clients := make([]*publicnetapi.PublicNetAPIClient, 0, len(regions))
	for _, region := range regions {
		endpoint, err := keystone.ResolveEndpointFromProvider(provider, publicNetServiceType, region)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve Public Net endpoint for region %q: %w", region, err)
		}
		cl, err := newAPIClient(endpoint, token)
		if err != nil {
			return nil, nil, fmt.Errorf("build Public Net client for region %q: %w", region, err)
		}
		clients = append(clients, cl)
	}
	return regions, clients, nil
}

// newAPIClient builds a Public Net API v1 client pinned to endpoint and token.
func newAPIClient(endpoint, token string) (*publicnetapi.PublicNetAPIClient, error) {
	return publicnetapi.NewPublicNetAPIClient(&publicnetapi.Config{
		URL:       endpoint,
		AuthToken: token,
	})
}

// ResolveProjectID returns the ID of the project the token is scoped to. It is
// used by the "create" command to fill the required project_id body field: the
// Public Net API is project-scoped, so the token already carries the target
// project. An empty result with a nil error means the token is not scoped to a
// project.
func ResolveProjectID(cmd *cobra.Command) (string, error) {
	provider, _, err := authenticatedProvider(cmd)
	if err != nil {
		return "", err
	}
	return keystone.ProjectIDFromProvider(provider)
}

// authenticatedProvider authenticates once against the identity endpoint and
// returns the authenticated ProviderClient plus the token that was used.
func authenticatedProvider(cmd *cobra.Command) (*gophercloud.ProviderClient, string, error) {
	token, err := resolveProjectToken(cmd)
	if err != nil {
		return nil, "", err
	}

	provider, err := keystone.Authenticate(keystone.Options{
		IdentityEndpoint: resolveEndpoint(cmd),
		Token:            token,
		Debug:            debugWriter(cmd),
	})
	if err != nil {
		return nil, "", err
	}
	return provider, token, nil
}

func debugWriter(cmd *cobra.Command) io.Writer {
	if debug, _ := cmd.Flags().GetBool(client.FlagDebug); debug {
		return cmd.ErrOrStderr()
	}
	return nil
}

// resolveProjectToken returns the API token from the "--token" flag, falling
// back to the persisted project-scoped token (when it has not expired) and
// then to the SLT_KS_TOKEN environment variable. The Public Net API lives on a
// Regional service but accepts a project-scoped token, so the project-scoped
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
