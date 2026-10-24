/*
Package authentication authenticates to the Selectel Keystone Identity v3
service and returns the issued API token.

Credentials are resolved with the gophercloud utils clientconfig package,
which reads them either from a clouds.yaml file (selected via the --cloud
profile, OS_CLOUD or the first configured entry) or from the OS_* environment
variables that an openrc file exports (OS_AUTH_URL, OS_USERNAME, OS_PASSWORD,
OS_PROJECT_NAME, OS_PROJECT_DOMAIN_NAME, ...).
*/
package authentication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"github.com/gophercloud/utils/v2/env"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
)

// defaultIdentityVersion is the Keystone major version used when neither
// clouds.yaml nor the OS_IDENTITY_API_VERSION variable pins one.
const defaultIdentityVersion = "3"

// Options configures how authentication credentials are resolved.
type Options struct {
	// Cloud is the name of the profile in clouds.yaml to use. When empty, the
	// OS_CLOUD environment variable is tried, then the first configured entry.
	Cloud string

	// AuthURL overrides the Keystone Identity v3 endpoint. When empty, it is
	// taken from clouds.yaml or OS_AUTH_URL.
	AuthURL string

	// EnvPrefix is the environment variable prefix expected by the openrc file.
	// The default is "OS_".
	EnvPrefix string

	// IdentityVersion is the major Keystone identity API version ("3"). When
	// empty it is guessed from the auth URL and the OS_IDENTITY_API_VERSION
	// variable, defaulting to 3.
	IdentityVersion string
}

// TokenResult carries the token issued by Keystone.
type TokenResult struct {
	// Token is the raw API token.
	Token string
	// ExpiresAt is the moment after which the token is no longer accepted by
	// the API.
	ExpiresAt time.Time
}

// Token authenticates to the Keystone Identity service with the credentials
// resolved from clouds.yaml or the environment and returns the issued token.
func Token(ctx context.Context, opts Options) (TokenResult, error) {
	authOpts, err := buildAuthOptions(opts)
	if err != nil {
		return TokenResult{}, err
	}

	provider, err := openstack.NewClient(authOpts.IdentityEndpoint)
	if err != nil {
		return TokenResult{}, err
	}

	if err := openstack.AuthenticateV3(ctx, provider, authOpts, gophercloud.EndpointOpts{}); err != nil {
		return TokenResult{}, err
	}

	result := TokenResult{Token: provider.Token()}
	if create, ok := provider.GetAuthResult().(tokens3.CreateResult); ok {
		if tok, err := create.ExtractToken(); err == nil {
			result.ExpiresAt = tok.ExpiresAt
		}
	}

	return result, nil
}

// buildAuthOptions resolves the gophercloud AuthOptions from clouds.yaml or
// the OS_* environment variables supplied by an openrc file. An explicit
// AuthURL in opts wins over both sources, and the identity version is forced
// to IdentityVersion (default "3") so V3 password authentication is used.
func buildAuthOptions(opts Options) (*gophercloud.AuthOptions, error) {
	envPrefix := opts.EnvPrefix
	if envPrefix == "" {
		envPrefix = "OS_"
	}

	clientOpts := &clientconfig.ClientOpts{
		Cloud:     opts.Cloud,
		AuthType:  clientconfig.AuthV3Password,
		EnvPrefix: envPrefix,
	}

	authOpts, err := clientconfig.AuthOptions(clientOpts)
	if err != nil {
		return nil, err
	}

	if opts.AuthURL != "" {
		authOpts.IdentityEndpoint = opts.AuthURL
	}

	authOpts.AllowReauth = false

	// Ensure the resolved identity endpoint is not empty.
	if authOpts.IdentityEndpoint == "" {
		authOpts.IdentityEndpoint = env.Getenv(envPrefix + "AUTH_URL")
	}
	if authOpts.IdentityEndpoint == "" {
		return nil, errors.New("no identity endpoint found: set OS_AUTH_URL, a clouds.yaml entry, or --auth-url")
	}

	identityVersion := opts.IdentityVersion
	if identityVersion == "" {
		identityVersion = defaultIdentityVersion
	}
	if identityVersion != defaultIdentityVersion {
		return nil, fmt.Errorf("unsupported identity version %q: only v3 is supported", identityVersion)
	}

	return authOpts, nil
}
