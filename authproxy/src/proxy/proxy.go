// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/authproxy/internal/httputil"
	"github.com/verrazzano/verrazzano/authproxy/src/apiserver"
	"github.com/verrazzano/verrazzano/authproxy/src/auth"
	"github.com/verrazzano/verrazzano/authproxy/src/config"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	callbackPath = "/_authentication_callback"
	logoutPath   = "/_logout"
)

var (
	getConfigFunc     = k8sutil.GetConfigFromController
	getOIDCConfigFunc = getOIDCConfiguration
)

var mutex sync.RWMutex

// AuthProxy wraps the server instance
type AuthProxy struct {
	http.Server
}

type handlerFuncType func(w http.ResponseWriter, r *http.Request)

// Handler performs HTTP handling for the AuthProxy Server
type Handler struct {
	URL           string
	Client        *retryablehttp.Client
	Log           *zap.SugaredLogger
	OIDCConfig    map[string]string
	Authenticator auth.Authenticator
	K8sClient     client.Client
	AuthInited    atomic.Bool
}

var _ http.Handler = &Handler{}

// InitializeProxy returns a configured AuthProxy instance
func InitializeProxy(port int) *AuthProxy {
	return &AuthProxy{
		Server: http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}
}

// ConfigureKubernetesAPIProxy configures the server handler and the proxy client for the AuthProxy instance
func ConfigureKubernetesAPIProxy(authproxy *AuthProxy, k8sClient client.Client, log *zap.SugaredLogger) error {
	restConfig, err := getConfigFunc()
	if err != nil {
		log.Errorf("Failed to get Kubeconfig for the proxy: %v", err)
		return err
	}

	rootCA, err := loadCAData(restConfig, log)
	if err != nil {
		return err
	}

	httpClient := httputil.GetHTTPClientWithCABundle(rootCA)
	authproxy.Handler = &Handler{
		URL:       restConfig.Host,
		Client:    httpClient,
		Log:       log,
		K8sClient: k8sClient,
	}
	return nil
}

// findPathHandler returns the path handler function given the request path
func (h *Handler) findPathHandler(req *http.Request) handlerFuncType {
	switch req.URL.Path {
	case callbackPath:
		return h.handleAuthCallback
	case logoutPath:
		return h.handleLogout
	default:
		return h.handleAPIRequest
	}
}

// ServeHTTP accepts an incoming server request and forwards it to the Kubernetes API server
func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.Log.Debug("Incoming request: %+v", httputil.ObfuscateRequestData(req))

	err := h.initializeAuthenticator()
	if err != nil {
		h.Log.Errorf("Failed to initialize Authenticator: %v", err)
		http.Error(rw, "Failed to initialize Authenticator", http.StatusInternalServerError)
		return
	}

	handlerFunc := h.findPathHandler(req)
	handlerFunc(rw, req)
}

// handleAuthCallback is the http handler for authentication callback
func (h *Handler) handleAuthCallback(rw http.ResponseWriter, req *http.Request) {
	// ingressHost := getIngressHost(req)
	// authenticator := auth.NewAuthenticator(createOidcConfig(ingressHost), h.Log, h.K8sClient)
	// authenticator.Verify(req, rw)
	// set cookies
}

// handleLogout is the http handler for logout
func (h *Handler) handleLogout(rw http.ResponseWriter, req *http.Request) {

}

// handleAPIRequest is the http handler for API requests
func (h *Handler) handleAPIRequest(rw http.ResponseWriter, req *http.Request) {
	apiRequest := apiserver.APIRequest{
		RW:            rw,
		Request:       req,
		Authenticator: h.Authenticator,
		Client:        h.Client,
		APIServerURL:  h.URL,
		CallbackPath:  callbackPath,
		Log:           h.Log,
	}
	apiRequest.ForwardAPIRequest()
}

// initializeAuthenticator initializes the handler authenticator
func (h *Handler) initializeAuthenticator() error {
	if h.AuthInited.Load() {
		return nil
	}

	oidcConfig := getOIDCConfigFunc()

	mutex.Lock()
	defer mutex.Unlock()

	// double-check the condition in case it changed by the time we acquired the lock
	if h.AuthInited.Load() {
		return nil
	}

	authenticator, err := auth.NewAuthenticator(&oidcConfig, h.Log, h.K8sClient)
	if err != nil {
		return err
	}
	h.Authenticator = authenticator
	h.AuthInited.Store(true)
	return nil
}

// loadCAData returns the config CA data from the byte array or from the file name
func loadCAData(config *rest.Config, log *zap.SugaredLogger) (*x509.CertPool, error) {
	if len(config.CAData) < 1 {
		rootCA, err := cert.NewPool(config.CAFile)
		if err != nil {
			log.Errorf("Failed to get in cluster Root Certificate for the Kubernetes API server")
		}
		return rootCA, err
	}

	rootCA, err := cert.NewPoolFromBytes(config.CAData)
	if err != nil {
		log.Errorf("Failed to load CA data from the Kubeconfig")
	}
	return rootCA, err
}

// getOIDCConfiguration returns an OIDC configuration populated from the config package
func getOIDCConfiguration() auth.OIDCConfiguration {
	return auth.OIDCConfiguration{
		ExternalURL: config.GetExternalURL(),
		ServiceURL:  config.GetServiceURL(),
		ClientID:    config.GetClientID(),
	}
}
