// Copyright (C) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	"net/http"
	"strings"
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
	OSDashboardsClient struct {
		httpClient *http.Client
		DoHTTP     func(request *http.Request) (*http.Response, error)
	}
)

func NewOSDashboardsClient(pas string) *OSDashboardsClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	od := &OSDashboardsClient{
		httpClient: &http.Client{Transport: tr},
	}
	od.DoHTTP = func(request *http.Request) (*http.Response, error) {
		request.SetBasicAuth("verrazzano", pas)
		return od.httpClient.Do(request)
	}
	return od
}

var defaultIndexPatterns = [...]string{VZSystemIndexPattern, VZAppIndexPattern}

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

// CreateDefaultIndexPatterns creates the defaultIndexPatterns in the OpenSearchDashboards if not existed
func (od *OSDashboardsClient) CreateDefaultIndexPatterns(log vzlog.VerrazzanoLogger, openSearchDashboardsEndpoint string) error {
	existingIndexPatterns, err := od.getDefaultIndexPatterns(openSearchDashboardsEndpoint, 50, fmt.Sprintf("%s+or+%s", strings.Replace(VZSystemIndexPattern, "*", "\\*", -1), strings.Replace(VZAppIndexPattern, "*", "\\*", -1)))
	if err != nil {
		zap.S().Infof("Isha error getting default index pattern", err)
		return err
	}
	var savedObjectPayloads []SavedObjectType
	for _, indexPattern := range defaultIndexPatterns {
		if existingIndexPatterns[indexPattern] {
			continue
		}
		savedObject := SavedObjectType{
			Type: IndexPattern,
			Attributes: Attributes{
				Title:         indexPattern,
				TimeFieldName: TimeStampField,
			},
		}
		savedObjectPayloads = append(savedObjectPayloads, savedObject)
	}
	if len(savedObjectPayloads) > 0 {
		log.Progressf("Creating default index patterns")
		err = od.creatIndexPatterns(log, savedObjectPayloads, openSearchDashboardsEndpoint)
		if err != nil {
			return err
		}
	}
	return nil
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
		log.Errorf("failed to create request for index patterns using bulk API %s", err.Error())
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("osd-xsrf", "true")
	resp, err := od.DoHTTP(req)
	if err != nil {
		log.Errorf("failed to create index patterns %s using bulk API %s", string(savedObjectBytes), err.Error())
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
