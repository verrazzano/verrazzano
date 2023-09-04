// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package auth

import (
	"net/http"

	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OIDCConfiguration struct {
	IssuerURL   string
	ClientID    string
	CallbackURL string
}

type Authenticator struct {
	oidcConfig OIDCConfiguration
	Log        *zap.SugaredLogger
	K8sClient  client.Client
}

func NewAuthenticator(oidcConfig OIDCConfiguration, log *zap.SugaredLogger, client client.Client) *Authenticator {
	return &Authenticator{oidcConfig: oidcConfig, Log: log, K8sClient: client}
}

// Authenticate authenticates the given request. If a redirect or error has been processed, then
// return true to indicate the request has been fully processed. Otherwise return false to indicate
// that request processing should continue
func (a Authenticator) Authenticate(req *http.Request, rw http.ResponseWriter) bool {
	// request is not processed
	return false
}
