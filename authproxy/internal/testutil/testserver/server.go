// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testProviderJSON struct {
	Issuer string `json:"issuer"`
}

// FakeOIDCProviderServer returns a server that handles requests to set up an OIDC provider
func FakeOIDCProviderServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		provider := testProviderJSON{
			Issuer: fmt.Sprintf("http://%s", req.Host),
		}

		providerJSON, err := json.Marshal(provider)
		assert.NoError(t, err)

		_, err = w.Write(providerJSON)
		assert.NoError(t, err)
	}))
}
