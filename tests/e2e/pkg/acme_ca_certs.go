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

func getACMEStagingCAs() [][]byte {
	letsEncryptStagingIntE1CA := loadStagingCA(newSimpleHTTPClient(), letsEncryptStagingIntE1, "E1")
	letsEncryptStagingIntR3CA := loadStagingCA(newSimpleHTTPClient(), letsEncryptStagingIntR3, "R3")
	return [][]byte{letsEncryptStagingIntE1CA, letsEncryptStagingIntR3CA}
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
	resp, err := doReq(resURL, "GET", "", "", "", "", nil, newRetryableHTTPClient(httpClient))
	if err != nil {
		Log(Error, fmt.Sprintf("Error loading ACME staging CA: %v", err))
<<<<<<< HEAD
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		Log(Error, fmt.Sprintf("Unable to load ACME %s staging CA, status: %v\n", caCertName, resp.StatusCode))
		return nil
	}
=======
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		Log(Error, fmt.Sprintf("Unable to load ACME %s staging CA, status: %v\n", caCertName, resp.StatusCode))
		return nil
	}
>>>>>>> 6ae4e52e... Updated projectCmd
	return resp.Body
}
