// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package pkg

import (
	"fmt"
	"net/http"
	"net/url"
)

const letsEncryptStagingIntR3 = "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-r3.pem"
const letsEncryptStagingIntE1 = "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-e1.pem"

type ACMEStagingCerts struct {
	letsEncryptStagingIntR3CA []byte
	letsEncryptStagingIntE1CA []byte
}

var stagingCerts = ACMEStagingCerts{
	letsEncryptStagingIntE1CA: loadStagingCA(newSimpleHTTPClient(), letsEncryptStagingIntE1, "E1"),
	letsEncryptStagingIntR3CA: loadStagingCA(newSimpleHTTPClient(), letsEncryptStagingIntR3, "R3"),
}

func getACMEStagingCAs() [][]byte {
	return [][]byte{stagingCerts.letsEncryptStagingIntE1CA, stagingCerts.letsEncryptStagingIntR3CA}
}

func newSimpleHTTPClient() *http.Client {
	tr := &http.Transport{}
	proxyURL := getProxyURL()
	if proxyURL != "" {
		tURL := url.URL{}
		tURLProxy, _ := tURL.Parse(proxyURL)
		tr.Proxy = http.ProxyURL(tURLProxy)
	}
	httpClient := &http.Client{Transport: tr}
	return httpClient
}

func loadStagingCA(httpClient *http.Client, resURL string, caCertName string) []byte {
	status, pemData := doGetWebPage(resURL, "", newRetryableHTTPClient(httpClient), "", "")
	if status < 200 || status > 299 {
		fmt.Printf("Unable to load ACME %s staging CA, status: %v\n", caCertName, status)
		return nil
	}
	return []byte(pemData)
}
