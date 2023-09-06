// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/verrazzano/verrazzano/pkg/certs"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ Authenticator = OIDCAuthenticator{}

type OIDCAuthenticator struct {
	oidcConfig *OIDCConfiguration
	Log        *zap.SugaredLogger
	K8sClient  client.Client
}

func (a OIDCAuthenticator) AuthenticateToken(ctx context.Context, token string) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (a OIDCAuthenticator) SetCallbackURL(url string) {
	// TODO implement me
	panic("implement me")
}

const (
	AuthTypeBasic  string = "Basic"
	AuthTypeBearer string = "Bearer"
)

// AuthHeader returns the authorization header on the request
func AuthHeader(req *http.Request) string {
	return req.Header.Get("Authorization")
}

func NewAuthenticator(oidcConfig *OIDCConfiguration, log *zap.SugaredLogger, client client.Client) *OIDCAuthenticator {
	return &OIDCAuthenticator{oidcConfig: oidcConfig, Log: log, K8sClient: client}
}

func (a OIDCAuthenticator) AuthenticateRequest(req *http.Request, rw http.ResponseWriter) (bool, error) {
	authHeader := req.Header.Get("Authorization")

	provider, _, err := a.CreateOIDCProvider(a.oidcConfig.ExternalURL)
	if err != nil {
		return true, fmt.Errorf("Failed to create OIDC provider for authentication: %v", err)
	}
	if authHeader == "" {
		err := a.performLoginRedirect(req, rw, provider)
		if err != nil {
			return true, fmt.Errorf("Could not redirect for login: %v", err)
		}
		// we either redirected or are sending an error - either way, request processing is done
		return true, nil
	}
	return false, nil
}

func (a OIDCAuthenticator) createContextWithHTTPClient() (context.Context, error) {
	caBundleData, err := certs.GetLocalClusterCABundleData(a.Log, a.K8sClient, context.TODO())
	if err != nil {
		return nil, err
	}
	var certPool *x509.CertPool = nil
	if caBundleData != nil {
		if certPool, err = cert.NewPoolFromBytes(caBundleData); err != nil {
			return nil, err
		}
	}
	httpClient := httputil.GetHTTPClientWithRootCA(certPool)
	ctx := context.Background()
	return context.WithValue(ctx, oauth2.HTTPClient, httpClient), nil
}

func VerifyAuth(req *http.Request) (int, error) {

	return http.StatusOK, nil
}

func (a OIDCAuthenticator) ToOIDCConfig() *oidc.Config {
	return &oidc.Config{
		ClientID: a.oidcConfig.ClientID,
		Now: func() time.Time {
			return time.Now()
		},
	}
}

// CreateOIDCProvider creates an OIDC Provider for the given configuration
func (a OIDCAuthenticator) CreateOIDCProvider(issuerURL string) (*oidc.Provider, context.Context, error) {
	ctx, err := a.createContextWithHTTPClient()
	if err != nil {
		return nil, nil, err
	}
	provider, err := oidc.NewProvider(ctx, issuerURL)
	return provider, ctx, err
}

func (a OIDCAuthenticator) performLoginRedirect(req *http.Request, rw http.ResponseWriter, provider *oidc.Provider) error {
	/* from LUA code
	local state = me.randomBase64(32)
	   local nonce = me.randomBase64(32)
	   local stateData = {
	       state = state,
	       request_uri = ngx.var.request_uri,
	       code_verifier = codeVerifier,
	       code_challenge = codeChallenge,
	       nonce = nonce
	   }
	   local redirectArgs = ngx.encode_args({
	       client_id = oidcClient,
	       response_type = 'code',
	       scope = 'openid',
	       code_challenge_method = 'S256',
	       code_challenge = codeChallenge,
	       state = state,
	       nonce = nonce,
	       redirect_uri = me.callbackUri
	   })
	   local redirectURL = me.getOidcAuthUri()..'?'..redirectArgs
	*/
	var state string
	var nonce string
	var err error
	if state, err = randomBase64(32); err != nil {
		return fmt.Errorf("Could not redirect for login - failed to generate random base64: %v", err)
	}
	if nonce, err = randomBase64(32); err != nil {
		return fmt.Errorf("Could not redirect for login - failed to generate random base64: %v", err)
	}

	oauthConfig := oauth2.Config{
		ClientID:    a.oidcConfig.ClientID,
		Endpoint:    provider.Endpoint(),
		RedirectURL: a.oidcConfig.CallbackURL,
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
	}
	http.Redirect(rw, req, oauthConfig.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
	return nil
}

func (a OIDCAuthenticator) Verify(req *http.Request, rw http.ResponseWriter) bool {
	// provider, ctx, err := a.CreateOIDCProvider(a.oidcConfig.ServiceURL)
	// if err != nil {
	// 	http.Error(rw, err.Error(), http.StatusInternalServerError)
	// }
	// verifier := provider.Verifier(a.ToOIDCConfig())
	// verifier.Verify(ctx, "")
	// TODO actually verify
	return true
}

func authType(authHeader string) string {
	authHeaderParts := strings.Split(authHeader, " ")
	if len(authHeaderParts) > 0 {
		return authHeaderParts[0]
	}
	return ""
}

func randomBase64(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

const authHeaderKey = "Authorization"

type OIDCAuthenticator struct {
	Log         *zap.SugaredLogger
	serviceURL  string
	externalURL string
	client      *retryablehttp.Client
	clientID    string
	verifier    atomic.Value
}

func NewAuthenticator(log *zap.SugaredLogger) (*OIDCAuthenticator, error) {
	authenticator := &OIDCAuthenticator{
		Log:         log,
		serviceURL:  "http://keycloak-http.keycloak.svc.cluster.local/auth/realms/verrazzano-system",
		externalURL: "https://keycloak.default.172.18.0.231.nip.io/auth/realms/verrazzano-system",
		client:      client.GetHTTPClientWithCABundle(&x509.CertPool{}),
		clientID:    "verrazzano-pg",
	}

	if err := authenticator.storeVerifier(); err != nil {
		log.Errorf("Failed to load verifier for the authenticator: %v", err)
		return nil, err
	}

	return authenticator, nil
}

func (a *OIDCAuthenticator) AuthenticateRequest(req *http.Request) (bool, error) {
	authHeader := strings.TrimSpace(req.Header.Get(authHeaderKey))
	splitHeader := strings.SplitN(authHeader, " ", 2)

	if len(splitHeader) < 2 || strings.ToLower(splitHeader[0]) != "bearer" {
		err := fmt.Errorf("failed to verify authorization bearer header")
		a.Log.Errorf("Failed to verify that authorization header had correct bearer format: %v", err)
		return false, err
	}

	return a.AuthenticateToken(req.Context(), splitHeader[1])
}

func (a *OIDCAuthenticator) AuthenticateToken(ctx context.Context, token string) (bool, error) {
	verifier := a.loadVerifier()

	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		a.Log.Errorf("Failed to verify JWT token: %v", err)
		return false, err
	}

	// Do issuer check for external URL
	// This is skipped in the go-oidc package because it could be the service or the ingress
	if idToken.Issuer != a.externalURL || idToken.Issuer != a.serviceURL {
		err := fmt.Errorf("failed to verify issuer, got %s, expected %s or %s", idToken.Issuer, a.serviceURL, a.externalURL)
		a.Log.Errorf("Failed to validate JWT issuer: %v", err)
		return false, err
	}

	return true, nil
}

func (a *OIDCAuthenticator) storeVerifier() error {
	provider, err := oidc.NewProvider(context.TODO(), a.serviceURL)
	if err != nil {
		a.Log.Errorf("Failed to load OIDC provider: %v", err)
		return err
	}

	config := &oidc.Config{
		ClientID:             a.clientID,
		SkipIssuerCheck:      true,
		SupportedSigningAlgs: []string{oidc.RS256},
		Now:                  time.Now,
	}

	verifier := provider.Verifier(config)
	a.verifier.Store(verifier)
	return nil
}

func (a *OIDCAuthenticator) loadVerifier() *oidc.IDTokenVerifier {
	verifier := a.verifier.Load()
	return verifier.(*oidc.IDTokenVerifier)
}
