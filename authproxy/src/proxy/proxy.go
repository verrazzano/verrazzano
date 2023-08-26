// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"go.uber.org/zap"
	"io"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	localClusterPrefix = "/clusters/local"

	kubernetesAPIServiceHostname = "kubernetes.default.svc.cluster.local"
)

type AuthProxy struct {
	http.Server
}

type Handler struct {
	Client *http.Client
	Log    *zap.SugaredLogger
}

func InitializeProxy() *AuthProxy {
	return &AuthProxy{
		Server: http.Server{
			Addr:         ":8777",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}
}

func ConfigureKubernetesAPIProxy(authproxy *AuthProxy, kubeconfig string, log *zap.SugaredLogger) error {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Errorf("Failed to get Kubeconfig for the proxy: %v", err)
	}

	transport := http.DefaultTransport
	transport.(*http.Transport).TLSClientConfig = &tls.Config{
		RootCAs:    common.CertPool(config.CAData),
		ServerName: kubernetesAPIServiceHostname,
		MinVersion: tls.VersionTLS12,
	}

	authproxy.Handler = Handler{
		Client: &http.Client{
			Timeout:   5 * time.Minute,
			Transport: transport,
		},
		Log: log,
	}
	return nil
}

func (h Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.Log.Debug("Incoming request: %+v", req)
	err := validateRequest(req)
	if err != nil {
		h.Log.Infof("Failed to validate request: %s", err.Error())
		return
	}

	reformattedReq, err := h.reformatAPIRequest(req)
	if err != nil {
		return
	}
	h.Log.Debug("Outgoing request: %+v", reformattedReq)

	resp, err := h.Client.Do(reformattedReq)
	if err != nil {
		h.Log.Errorf("Failed to send request: %v", err)
		return
	}
	if resp == nil {
		h.Log.Errorf("Empty response from server: %v", err)
		return
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			h.Log.Errorf("Failed to close response body: %v", err)
		}
	}()

	_, err = io.Copy(rw, resp.Body)
	if err != nil {
		h.Log.Errorf("Failed to send request: %v", err)
	}
}

func (h Handler) reformatAPIRequest(req *http.Request) (*http.Request, error) {
	formattedReq := req.Clone(context.TODO())
	formattedReq.Host = kubernetesAPIServiceHostname
	formattedReq.RequestURI = ""

	path := strings.Replace(req.URL.Path, localClusterPrefix, "", 1)
	formattedURL, err := url.Parse(fmt.Sprintf("%s%s", kubernetesAPIServiceHostname, path))
	if err != nil {
		h.Log.Errorf("Failed to format incoming url: %v", err)
		return nil, err
	}
	formattedReq.URL = formattedURL

	return formattedReq, nil
}

func validateRequest(req *http.Request) error {
	if !strings.HasPrefix(req.URL.String(), localClusterPrefix) {
		return fmt.Errorf("request url: '%v' does not have cluster path", req.URL)
	}
	return nil
}
