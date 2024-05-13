// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package pkg

import (
	"fmt"
	"net/http"
)

const letsEncryptStagingIntR10 = "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-r10.pem"
const letsEncryptStagingIntE5 = "https://letsencrypt.org/certs/staging/letsencrypt-stg-int-e5.pem"

func getACMEStagingCAs() [][]byte {
	letsEncryptStagingIntE5CA := loadStagingCA(newSimpleHTTPClient(), letsEncryptStagingIntE5, "E5")
	letsEncryptStagingIntR10CA := loadStagingCA(newSimpleHTTPClient(), letsEncryptStagingIntR10, "R10")
	return [][]byte{letsEncryptStagingIntE5CA, letsEncryptStagingIntR10CA}
}

func newSimpleHTTPClient() *http.Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	httpClient := &http.Client{Transport: tr}
	return httpClient
}

func loadStagingCA(httpClient *http.Client, resURL string, caCertName string) []byte {
	resp, err := doReq(resURL, "GET", "", "", "", "", nil, newRetryableHTTPClient(httpClient))
	if err != nil {
		Log(Error, fmt.Sprintf("Error loading ACME staging CA: %v", err))
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		Log(Error, fmt.Sprintf("Unable to load ACME %s staging CA, status: %v\n", caCertName, resp.StatusCode))
		return nil
	}
	return resp.Body
}
