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

	// Scope selects the token scope. ScopeProject (the default) requests a
	// project-scoped token, which is what services such as MKS and CRaaS accept;
	// ScopeAccount requests an account- (domain-) scoped token that unlocks
	// account-level management APIs.
	Scope Scope
}

// TokenResult carries the token issued by Keystone.
type TokenResult struct {
	// Token is the raw API token.
	Token string
	// IssuedAt is the moment the token was issued.
	IssuedAt time.Time
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
		// gophercloud's ExtractToken exposes only the ID and expiry; the issue
		// timestamp must be pulled from the raw response body.
		if issued, err := extractIssuedAt(create); err == nil {
			result.IssuedAt = issued
		}
	}

	return result, nil
}

// extractIssuedAt reads the "issued_at" timestamp from the raw token response
// body. It returns an error (ignored by the caller, which keeps the timestamp
// at the zero time) when the field is absent or malformed.
func extractIssuedAt(create tokens3.CreateResult) (time.Time, error) {
	body, ok := create.Body.(map[string]any)
	if !ok {
		return time.Time{}, errors.New("token response body is not an object")
	}
	token, ok := body["token"].(map[string]any)
	if !ok {
		return time.Time{}, errors.New("token response has no token object")
	}
	raw, ok := token["issued_at"].(string)
	if !ok {
		return time.Time{}, errors.New("token response has no issued_at")
	}
	return time.Parse(time.RFC3339, raw)
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

	if err := applyScope(authOpts, opts.Scope); err != nil {
		return nil, err
	}

	return authOpts, nil
}

// applyScope pins the AuthOptions to the requested token scope. A project
// scope keeps whatever scope clientconfig derived (it already yields a project
// scope when a project is configured) but insists that a project is available.
// An account scope requests a domain-scoped token, reusing the user's domain
// (the Selectel account ID) even when a project is configured.
func applyScope(authOpts *gophercloud.AuthOptions, scope Scope) error {
	switch scope {
	case ScopeAccount:
		domainID, domainName := authOpts.DomainID, authOpts.DomainName
		if domainID == "" && domainName == "" {
			return errors.New("account scope requires a user domain (the account ID): set OS_USER_DOMAIN_NAME or OS_DOMAIN_NAME so it can be resolved")
		}
		authOpts.Scope = &gophercloud.AuthScope{
			DomainID:   domainID,
			DomainName: domainName,
		}
		// Clear any project scope so the account scope is not overridden.
		authOpts.TenantID = ""
		authOpts.TenantName = ""
	case ScopeProject:
		hasProject := authOpts.TenantID != "" || authOpts.TenantName != ""
		if !hasProject && authOpts.Scope != nil {
			hasProject = authOpts.Scope.ProjectID != "" || authOpts.Scope.ProjectName != ""
		}
		if !hasProject {
			return errors.New("project scope requires a project: set OS_PROJECT_NAME (with OS_PROJECT_DOMAIN_NAME) or configure a project in clouds.yaml")
		}
	default:
		// Empty scope keeps clientconfig's default behavior (project-scoped
		// when a project is present, otherwise unscoped).
	}
	return nil
}
