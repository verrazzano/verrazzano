// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/authproxy/internal/httputil"
	"github.com/verrazzano/verrazzano/authproxy/src/auth"
	"github.com/verrazzano/verrazzano/authproxy/src/cors"
	"go.uber.org/zap"
)

const (
	localClusterPrefix          = "/clusters/local"
	kubernetesAPIServerHostname = "kubernetes.default.svc.cluster.local"
	contentTypeHeader           = "Content-Type"
)

// APIRequest stores the data necessary to make a request to the API server
type APIRequest struct {
	RW            http.ResponseWriter
	Request       *http.Request
	Authenticator auth.Authenticator
	Client        *retryablehttp.Client
	APIServerURL  string
	CallbackPath  string
	Log           *zap.SugaredLogger
}

// ForwardAPIRequest forwards a given API request to the API server
func (a APIRequest) ForwardAPIRequest() {
	rw := a.RW
	req := a.Request

	err := validateRequest(req)
	if err != nil {
		a.Log.Debugf("Failed to validate request: %s", err.Error())
		http.Error(rw, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	ingressHost := getIngressHost(req)
	if statusCode, err := cors.AddCORSHeaders(req, rw, ingressHost); err != nil {
		http.Error(rw, err.Error(), statusCode)
		return
	}

	if req.Method == http.MethodOptions {
		rw.Header().Set("Content-Length", "0")
		rw.WriteHeader(http.StatusOK)
		return
	}

	a.Authenticator.SetCallbackURL(fmt.Sprintf("https://%s%s", ingressHost, a.CallbackPath))
	continueProcessing, err := a.Authenticator.AuthenticateRequest(req, rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusUnauthorized)
		return
	}
	if !continueProcessing {
		http.Error(rw, "request was not authenticated", http.StatusUnauthorized)
		return
	}

	reformattedReq, err := a.reformatAPIRequest(req)
	if err != nil {
		http.Error(rw, "Failed to reformat request for the Kubernetes API server", http.StatusUnprocessableEntity)
		return
	}
	a.Log.Debug("Outgoing request: %+v", httputil.ObfuscateRequestData(reformattedReq.Request))

	resp, err := a.Client.Do(reformattedReq)
	if err != nil {
		errResponse := fmt.Sprintf("Failed to forward request to the Kubernetes API server: %s", err.Error())
		http.Error(rw, errResponse, http.StatusBadRequest)
		return
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			a.Log.Errorf("Failed to close response body: %v", err)
		}
	}()

	var responseBody = io.NopCloser(strings.NewReader(""))
	if resp != nil {
		responseBody = resp.Body
	}

	if _, ok := resp.Header[contentTypeHeader]; ok {
		for _, h := range resp.Header[contentTypeHeader] {
			rw.Header().Set(contentTypeHeader, h)
		}
	} else {
		bodyData, err := io.ReadAll(responseBody)
		if err != nil {
			a.Log.Errorf("Failed to read response body for content type detection: %v", err)
			return
		}

		rw.Header().Set(contentTypeHeader, http.DetectContentType(bodyData))
	}

	_, err = io.Copy(rw, responseBody)
	if err != nil {
		a.Log.Errorf("Failed to copy server response to read writer: %v", err)
		return
	}
}

// reformatAPIRequest reformats an incoming HTTP request to be sent to the Kubernetes API Server
func (a APIRequest) reformatAPIRequest(req *http.Request) (*retryablehttp.Request, error) {
	formattedReq := req.Clone(context.TODO())
	formattedReq.Host = kubernetesAPIServerHostname
	formattedReq.RequestURI = ""

	path := strings.Replace(req.URL.Path, localClusterPrefix, "", 1)
	newReq, err := url.JoinPath(a.APIServerURL, path)
	if err != nil {
		a.Log.Errorf("Failed to format request path for path %s: %v", path, err)
		return nil, err
	}

	formattedURL, err := url.Parse(newReq)
	if err != nil {
		a.Log.Errorf("Failed to format incoming url: %v", err)
		return nil, err
	}
	formattedURL.RawQuery = req.URL.RawQuery
	formattedReq.URL = formattedURL

	retryableReq, err := retryablehttp.FromRequest(formattedReq)
	if err != nil {
		a.Log.Errorf("Failed to convert reformatted request to a retryable request: %v", err)
		return retryableReq, err
	}

	return retryableReq, nil
}

// getIngressHost determines the ingress host from the request headers
func getIngressHost(req *http.Request) string {
	if host := req.Header.Get("x-forwarded-host"); host != "" {
		return host
	}
	if host := req.Header.Get("host"); host != "" {
		return host
	}
	if host := req.Host; host != "" {
		return host
	}
	return "invalid-hostname"
}

// validateRequest performs request validation before the request is processed
func validateRequest(req *http.Request) error {
	if !strings.HasPrefix(req.URL.Path, localClusterPrefix) {
		return fmt.Errorf("request path: '%v' does not have expected cluster path, i.e. '/clusters/local/api/v1'", req.URL.Path)
	}
	return nil
}
