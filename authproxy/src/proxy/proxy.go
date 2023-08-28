// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
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
	Client *http.Client
	Log    *zap.SugaredLogger
}

var _ http.Handler = Handler{}

// InitializeProxy returns a configured AuthProxy instance
func InitializeProxy() *AuthProxy {
	return &AuthProxy{
		Server: http.Server{
			Addr:         ":8777",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
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

	transport := http.DefaultTransport
	transport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec //#gosec G101
	}

	authproxy.Handler = Handler{
		URL: config.Host,
		Client: &http.Client{
			Timeout:   time.Minute,
			Transport: transport,
		},
		Log: log,
	}
	return nil
}

// ServeHTTP accepts an incoming server request and forwards it to the Kubernetes API server
func (h Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.Log.Debug("Incoming request: %+v", req)
	err := validateRequest(req)
	if err != nil {
		h.Log.Debugf("Failed to validate request: %s", err.Error())
		http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	reformattedReq, err := h.reformatAPIRequest(req)
	if err != nil {
		http.Error(rw, "Failed to reformat request for the Kubernetes API server", http.StatusUnprocessableEntity)
		return
	}
	h.Log.Debug("Outgoing request: %+v", reformattedReq)

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

// reformatAPIRequest reformats an incoming HTTP request to be sent to the Kubernetes API Server
func (h Handler) reformatAPIRequest(req *http.Request) (*http.Request, error) {
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

	return formattedReq, nil
}

// validateRequest performs request validation before the request is processed
func validateRequest(req *http.Request) error {
	if !strings.HasPrefix(req.URL.Path, localClusterPrefix) {
		return fmt.Errorf("request path: '%v' does not have expected cluster path, i.e. '/clusters/local/api/v1'", req.URL.Path)
	}
	return nil
}
