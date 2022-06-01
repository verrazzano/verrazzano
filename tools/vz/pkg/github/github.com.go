// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package github

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
)

// ReleaseAsset - subset of a GitHub release asset
type ReleaseAsset struct {
	TagName string `json:"tag_name"`
}

// newClient - create an HTTP client for accessing GitHub
func newClient() (*http.Client, error) {
	tr := &http.Transport{}
	httpClient := &http.Client{Transport: tr}

	return httpClient, nil
}

// ListReleases - return the list of Verrazzano releases
func ListReleases() ([]string, error) {
	var releaseTags []string

	// Create HTTP client
	client, err := newClient()
	if err != nil {
		return releaseTags, err
	}

	// Create the list request
	var buf io.ReadWriter
	request, err := http.NewRequest(http.MethodGet, constants.VerrazzanoReleaseList, buf)
	if err != nil {
		return releaseTags, err
	}
	request.Header.Set("Accept", "application/json")

	// Get the list of releases
	resp, err := client.Do(request)
	if err != nil {
		return releaseTags, err
	}

	// Decode the body to the list of releases
	defer resp.Body.Close()
	var releases []ReleaseAsset
	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return releaseTags, err
	}

	// Populate the return list
	for _, release := range releases {
		releaseTags = append(releaseTags, release.TagName)
	}

	return releaseTags, nil
}
