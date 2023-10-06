// Copyright (C) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

// VZSystemIndexPattern for Verrazzano System default index pattern
const (
	VZSystemIndexPattern = "verrazzano-system"
	VZAppIndexPattern    = "verrazzano-application*"
	IndexPattern         = "index-pattern"

	// TimeStamp used to add timestamp as TimeFieldName in the index pattern
	TimeStampField = "@timestamp"
)

type (
	IndexPatterns struct {
		Total        int           `json:"total"`
		Page         int           `json:"page"`
		SavedObjects []SavedObject `json:"saved_objects,omitempty"`
	}

	SavedObject struct {
		ID         string `json:"id"`
		Attributes `json:"attributes"`
	}

	Attributes struct {
		Title         string `json:"title"`
		TimeFieldName string `json:"timeFieldName,omitempty"`
	}
)

// SavedObjectType specifies the OpenSearch SavedObject including index-patterns.
type SavedObjectType struct {
	Type       string `json:"type"`
	Attributes `json:"attributes"`
}

// creatIndexPatterns creates the given IndexPattern in the OpenSearch-Dashboards by calling bulk API.
func (od *OSDashboardsClient) creatIndexPatterns(log vzlog.VerrazzanoLogger, savedObjectList []SavedObjectType, openSearchDashboardsEndpoint string) error {
	savedObjectBytes, err := json.Marshal(savedObjectList)
	if err != nil {
		return err
	}
	indexPatternURL := fmt.Sprintf("%s/api/saved_objects/_bulk_create", openSearchDashboardsEndpoint)
	req, err := http.NewRequest("POST", indexPatternURL, strings.NewReader(string(savedObjectBytes)))
	if err != nil {
		log.Errorf("Failed to create request for index patterns using bulk API %s", err.Error())
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("osd-xsrf", "true")
	resp, err := od.DoHTTP(req)
	if err != nil {
		log.Errorf("Failed to create index patterns %s using bulk API %s", string(savedObjectBytes), err.Error())
		return fmt.Errorf("failed to post index patterns in OpenSearch dashboards: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("post status code %d when creating index patterns: %s", resp.StatusCode, string(savedObjectBytes))
	}
	return nil
}

// getDefaultIndexPatterns fetches the existing defaultIndexPatterns.
func (od *OSDashboardsClient) getDefaultIndexPatterns(openSearchDashboardsEndpoint string, perPage int, searchQuery string) (map[string]bool, error) {
	defaultIndexPatternMap := map[string]bool{}
	savedObjects, err := od.getPatterns(openSearchDashboardsEndpoint, perPage, searchQuery)
	if err != nil {
		return defaultIndexPatternMap, err
	}
	for _, savedObject := range savedObjects {
		if isDefaultIndexPattern(savedObject.Title) {
			defaultIndexPatternMap[savedObject.Title] = true
		}
	}
	return defaultIndexPatternMap, nil
}

// isDefaultIndexPattern checks whether given index pattern is default index pattern or not
func isDefaultIndexPattern(indexPattern string) bool {
	for _, defaultIndexPattern := range defaultIndexPatterns {
		if defaultIndexPattern == indexPattern {
			return true
		}
	}
	return false
}
func (od *OSDashboardsClient) getPatterns(dashboardsEndPoint string, perPage int, searchQuery string) ([]SavedObject, error) {
	var savedObjects []SavedObject
	currentPage := 1

	// Index Pattern is a paginated response type, so we need to page out all data
	for {
		url := fmt.Sprintf("%s/api/saved_objects/_find?type=index-pattern&fields=title&per_page=%d&page=%d", dashboardsEndPoint, perPage, currentPage)
		if searchQuery != "" {
			url = fmt.Sprintf("%s/api/saved_objects/_find?search=%s&type=index-pattern&fields=title&per_page=%d&page=%d", dashboardsEndPoint, searchQuery, perPage, currentPage)
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		resp, err := od.DoHTTP(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("got code %d when querying index patterns", resp.StatusCode)
		}
		indexPatterns := &IndexPatterns{}
		if err := json.NewDecoder(resp.Body).Decode(indexPatterns); err != nil {
			return nil, fmt.Errorf("failed to decode index pattern response body: %v", err)
		}
		currentPage++
		savedObjects = append(savedObjects, indexPatterns.SavedObjects...)
		// paginate responses until we have all the index patterns
		if len(savedObjects) >= indexPatterns.Total {
			break
		}
	}

	return savedObjects, nil
}
