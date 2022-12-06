// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"io"
)

// HTTPGet issues an HTTP GET request with basic auth to the specified URL. httpGet returns the HTTP status code
// and an error.
func HTTPGet(url string, httpClient *retryablehttp.Client, credentials *pkg.UsernamePassword) (int, error) {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(credentials.Username, credentials.Password)
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()

	return resp.StatusCode, nil
}

func GetMetricLabels(_ string) map[string]string {
	return map[string]string{
		//"app_oam_dev_component": podName,
		"verrazzano_cluster": "local",
	}
}
