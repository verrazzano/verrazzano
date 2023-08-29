// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
)

const (
	localClusterPrefix = "/clusters/local"

	kubernetesAPIServerHostname = "kubernetes.default.svc.cluster.local"
)

var getConfigFunc = k8sutil.GetConfigFromController

// AuthProxy wraps the server instance
type AuthProxy struct {
	http.Server
}

// Handler performs HTTP handling for the AuthProxy Server
type Handler struct {
	URL    string
	Client *retryablehttp.Client
	Log    *zap.SugaredLogger
}

var _ http.Handler = Handler{}

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
func ConfigureKubernetesAPIProxy(authproxy *AuthProxy, log *zap.SugaredLogger) error {
	config, err := getConfigFunc()
	if err != nil {
		log.Errorf("Failed to get Kubeconfig for the proxy: %v", err)
		return err
	}

	rootCA, err := loadCAData(config, log)
	if err != nil {
		return err
	}

	transport := http.DefaultTransport
	transport.(*http.Transport).TLSClientConfig = &tls.Config{
		RootCAs:    rootCA,
		MinVersion: tls.VersionTLS12,
	}

	client := retryablehttp.NewClient()
	client.HTTPClient.Transport = transport

	authproxy.Handler = Handler{
		URL:    config.Host,
		Client: client,
		Log:    log,
	}
	return nil
}

// ServeHTTP accepts an incoming server request and forwards it to the Kubernetes API server
func (h Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.Log.Debug("Incoming request: %+v", obfuscateRequestData(req))
	err := validateRequest(req)

	if err != nil {
		h.Log.Debugf("Failed to validate request: %s", err.Error())
		http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	ingressHost := getIngressHost(req)
	if statusCode, err := addCORSHeaders(req, rw, ingressHost); err != nil {
		http.Error(rw, err.Error(), statusCode)
		return
	}

	if req.Method == http.MethodOptions {
		if statusCode, err := handleOptionsRequest(req, rw); err != nil {
			http.Error(rw, err.Error(), statusCode)
		}
		return
	}

	if statusCode, err := handleAuth(req, rw); err != nil {
		http.Error(rw, err.Error(), statusCode)
		return
	}

	reformattedReq, err := h.reformatAPIRequest(req)
	if err != nil {
		http.Error(rw, "Failed to reformat request for the Kubernetes API server", http.StatusUnprocessableEntity)
		return
	}
	h.Log.Debug("Outgoing request: %+v", obfuscateRequestData(reformattedReq.Request))

	resp, err := h.Client.Do(reformattedReq)
	if err != nil {
		errResponse := fmt.Sprintf("Failed to forward request to the Kubernetes API server: %s", err.Error())
		http.Error(rw, errResponse, http.StatusBadRequest)
		return
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			h.Log.Errorf("Failed to close response body: %v", err)
		}
	}()

	var responseBody = io.NopCloser(strings.NewReader(""))
	if resp != nil {
		responseBody = resp.Body
	}

	_, err = io.Copy(rw, responseBody)
	if err != nil {
		h.Log.Errorf("Failed to copy server response to read writer: %v", err)
	}
}

func handleOptionsRequest(req *http.Request, rw http.ResponseWriter) (int, error) {
	return http.StatusOK, nil
}

func addCORSHeaders(req *http.Request, rw http.ResponseWriter, ingressHost string) (int, error) {
	// TODO get origin header, check if it is an allowed origin, add CORS response headers
	return http.StatusOK, nil
}

func handleAuth(req *http.Request, rw http.ResponseWriter) (int, error) {
	authHeader := req.Header.Get("Authorization")
	if authHeader == "" {
		// TODO Handle callback/logout cases and if needed, perform authentication flow
	}
	return http.StatusOK, nil
}

// getIngressHost determines the ingress host from the request headers
func getIngressHost(req *http.Request) string {
	if host := req.Header.Get("x-forwarded-host"); host != "" {
		return host
	}
	if host := req.Header.Get("host"); host != "" {
		return host
	}
	return "invalid-hostname"
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

// reformatAPIRequest reformats an incoming HTTP request to be sent to the Kubernetes API Server
func (h Handler) reformatAPIRequest(req *http.Request) (*retryablehttp.Request, error) {
	formattedReq := req.Clone(context.TODO())
	formattedReq.Host = kubernetesAPIServerHostname
	formattedReq.RequestURI = ""

	path := strings.Replace(req.URL.Path, localClusterPrefix, "", 1)
	newReq, err := url.JoinPath(h.URL, path)
	if err != nil {
		h.Log.Errorf("Failed to format request path for path %s: %v", path, err)
		return nil, err
	}

	formattedURL, err := url.Parse(newReq)
	if err != nil {
		h.Log.Errorf("Failed to format incoming url: %v", err)
		return nil, err
	}
	formattedReq.URL = formattedURL

	retryableReq, err := retryablehttp.FromRequest(formattedReq)
	if err != nil {
		h.Log.Errorf("Failed to convert reformatted request to a retryable request: %v", err)
		return retryableReq, err
	}

	return retryableReq, nil
}

// validateRequest performs request validation before the request is processed
func validateRequest(req *http.Request) error {
	if !strings.HasPrefix(req.URL.Path, localClusterPrefix) {
		return fmt.Errorf("request path: '%v' does not have expected cluster path, i.e. '/clusters/local/api/v1'", req.URL.Path)
	}
	return nil
}

// obfuscateRequestData removes the Authorization header data from the request before logging
func obfuscateRequestData(req *http.Request) *http.Request {
	hiddenReq := req.Clone(context.TODO())
	authKey := "Authorization"
	for i := range hiddenReq.Header[authKey] {
		hiddenReq.Header[authKey][i] = vzpassword.MaskFunction("")(hiddenReq.Header[authKey][i])
	}
	return hiddenReq
}
