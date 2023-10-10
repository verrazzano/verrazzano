// Copyright (C) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

type (
	OSDashboardsClient struct {
		httpClient *http.Client
		DoHTTP     func(request *http.Request) (*http.Response, error)
	}
)

func NewOSDashboardsClient(pas string) *OSDashboardsClient {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec //#gosec G402
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

// CreateDefaultIndexPatterns creates the defaultIndexPatterns in the OpenSearchDashboards if not existed
func (od *OSDashboardsClient) CreateDefaultIndexPatterns(log vzlog.VerrazzanoLogger, openSearchDashboardsEndpoint string) error {
	existingIndexPatterns, err := od.getDefaultIndexPatterns(openSearchDashboardsEndpoint, 50, fmt.Sprintf("%s+or+%s", strings.Replace(VZSystemIndexPattern, "*", "\\*", -1), strings.Replace(VZAppIndexPattern, "*", "\\*", -1)))
	if err != nil {
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
