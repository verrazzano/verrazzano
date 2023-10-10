// Copyright (C) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

// TestCreateDefaultIndexPatterns tests the CreateDefaultIndexPatterns to verify the creation of default index patterns
func TestCreateDefaultIndexPatterns(t *testing.T) {
	type fields struct {
		httpClient *http.Client
		DoHTTP     func(request *http.Request) (*http.Response, error)
	}
	type args struct {
		openSearchDashboardsEndpoint string
	}
	emptyIndexPattern := IndexPatterns{
		SavedObjects: []SavedObject{},
	}
	indexPatterns := IndexPatterns{
		SavedObjects: []SavedObject{
			{
				ID: "ID1",
				Attributes: Attributes{
					Title: VZAppIndexPattern,
				},
			},
			{
				ID: "ID2",
				Attributes: Attributes{
					Title: VZSystemIndexPattern,
				},
			},
		},
	}
	emptyResponseBody, err := json.Marshal(emptyIndexPattern)
	if err != nil {
		t.Errorf("Json marshalling error")
	}
	responseBody, err := json.Marshal(indexPatterns)
	if err != nil {
		t.Errorf("Json marshalling error")
	}
	arg := args{
		"testAddress",
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// GIVEN OSDashboardsClient
		// WHEN CreateDefaultIndexPatterns is called
		// THEN default index patterns are created if they don't exist.
		{
			"TestCreateDefaultIndexPatterns when default index patterns don't exist",
			fields{
				httpClient: http.DefaultClient,
				DoHTTP: func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(string(emptyResponseBody))),
					}, nil
				},
			},
			arg,
			false,
		},
		// GIVEN OSDashboardsClient
		// WHEN CreateDefaultIndexPatterns is called
		// THEN default index patterns are not created if they exist.
		{
			"TestCreateDefaultIndexPatterns when default index patterns exist",
			fields{
				httpClient: http.DefaultClient,
				DoHTTP: func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(string(responseBody))),
					}, nil
				},
			},
			arg,
			false,
		},
		// GIVEN OSDashboardsClient
		// WHEN CreateDefaultIndexPatterns is called
		// THEN error is returned if index pattern API fails.
		{
			"TestCreateDefaultIndexPatterns when error occurs during index pattern API call",
			fields{
				httpClient: http.DefaultClient,
				DoHTTP: func(request *http.Request) (*http.Response, error) {
					return nil, fmt.Errorf("internal server error")
				},
			},
			arg,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			od := &OSDashboardsClient{
				httpClient: tt.fields.httpClient,
				DoHTTP:     tt.fields.DoHTTP,
			}
			if err = od.CreateDefaultIndexPatterns(vzlog.DefaultLogger(), tt.args.openSearchDashboardsEndpoint); (err != nil) != tt.wantErr {
				t.Errorf("CreateDefaultIndexPatterns() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDefaultIndexPattern test defaultIndexPattern function to check whether given string is default index pattern or not
// GIVEN a string
// WHEN defaultIndexPattern is called
// THEN true is returned if given string is a default index pattern, else false is returned.
func TestDefaultIndexPattern(t *testing.T) {
	type args struct {
		indexPattern string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"TestDefaultIndexPattern when given string is default index pattern",
			args{VZSystemIndexPattern},
			true,
		},
		{
			"TestDefaultIndexPattern when given string is default index pattern",
			args{"pattern1"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDefaultIndexPattern(tt.args.indexPattern); got != tt.want {
				t.Errorf("isDefaultIndexPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}
