// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package getlogs

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeHTTP struct {
	resp        *http.Response
	httpDoError error
}

type fakeBody struct {
	bodyData      string
	httpReadError error
}

var _ httpClientI = &fakeHTTP{}

// TestGetters tests the worker getters
// GIVEN a worker
//
//	WHEN the getter methods are calls
//	THEN ensure that the correct results are returned
func TestGetters(t *testing.T) {
	w, err := NewGetLogsWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypeGetLogs, wd.WorkerType)
	assert.Equal(t, "The log getter worker performs GET requests on the OpenSearch endpoint", wd.Description)
	assert.Equal(t, config.WorkerTypeGetLogs, wd.MetricsName)

	el := w.GetEnvDescList()
	assert.Len(t, el, 0)

	logged := w.WantLoopInfoLogged()
	assert.False(t, logged)
}

func TestGetMetricDescList(t *testing.T) {
	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "opensearch_get_success_count_total", help: "The total number of successful OpenSearch GET requests"},
		{name: "2", fqName: "opensearch_get_failure_count_total", help: "The total number of successful OpenSearch GET requests"},
		{name: "3", fqName: "opensearch_get_success_latency_nanoseconds", help: "The latency of successful OpenSearch GET requests in nanoseconds"},
		{name: "4", fqName: "opensearch_get_failure_latency_nanoseconds", help: "The latency of failed OpenSearch GET requests in nanoseconds"},
		{name: "5", fqName: "opensearch_get_data_chars_total", help: "The total number of characters return from OpenSearch get request"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewGetLogsWorker()
			assert.NoError(t, err)
			w := wi.(worker)
			assert.NoError(t, err)
			dl := w.GetMetricDescList()
			var found int
			for _, d := range dl {
				s := d.String()
				if strings.Contains(s, test.fqName) && strings.Contains(s, test.help) {
					found++
				}
			}
			assert.Equal(t, 1, found)
		})
	}
}

func TestGetMetricList(t *testing.T) {
	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "opensearch_get_success_count_total", help: "The total number of successful OpenSearch GET requests"},
		{name: "2", fqName: "opensearch_get_failure_count_total", help: "The total number of successful OpenSearch GET requests"},
		{name: "3", fqName: "opensearch_get_success_latency_nanoseconds", help: "The latency of successful OpenSearch GET requests in nanoseconds"},
		{name: "4", fqName: "opensearch_get_failure_latency_nanoseconds", help: "The latency of failed OpenSearch GET requests in nanoseconds"},
		{name: "5", fqName: "opensearch_get_data_chars_total", help: "The total number of characters return from OpenSearch get request"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewGetLogsWorker()
			w := wi.(worker)
			assert.NoError(t, err)
			ml := w.GetMetricList()
			var found int
			for _, m := range ml {
				s := m.Desc().String()
				if strings.Contains(s, test.fqName) && strings.Contains(s, test.help) {
					found++
				}
			}
			assert.Equal(t, 1, found)
		})
	}
}

// TestDoWork tests the DoWork method
// GIVEN a worker
//
//	WHEN the DoWork methods is called
//	THEN ensure that the correct results are returned
func TestDoWork(t *testing.T) {
	tests := []struct {
		name         string
		doworkError  error
		httpDoError  error
		nilResp      bool
		statusCode   int
		reqCount     int
		successCount int
		failureCount int
		fakeBody
	}{
		{
			name:         "1",
			statusCode:   200,
			reqCount:     1,
			successCount: 1,
			failureCount: 0,
			fakeBody: fakeBody{
				bodyData: "bodydata",
			},
		},
		{
			name:         "2",
			reqCount:     1,
			successCount: 0,
			failureCount: 1,
			doworkError:  errors.New("error"),
			httpDoError:  errors.New("error"),
			fakeBody: fakeBody{
				bodyData: "bodydata",
			},
		},
		{
			name:         "3",
			statusCode:   500,
			doworkError:  errors.New("error"),
			reqCount:     1,
			successCount: 0,
			failureCount: 1,
			fakeBody: fakeBody{
				bodyData:      "bodydata",
				httpReadError: errors.New("error"),
			},
		},
		{
			name:         "4",
			doworkError:  errors.New("GET request to endpoint received a nil response"),
			nilResp:      true,
			reqCount:     1,
			successCount: 0,
			failureCount: 1,
			fakeBody: fakeBody{
				bodyData:      "bodydata",
				httpReadError: errors.New("error"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := httpClient
			defer func() {
				httpClient = f
			}()
			var resp *http.Response
			if !test.nilResp {
				resp = &http.Response{
					StatusCode:    test.statusCode,
					Body:          &test.fakeBody,
					ContentLength: int64(len(test.bodyData)),
				}
			}
			httpClient = &fakeHTTP{
				httpDoError: test.doworkError,
				resp:        resp,
			}

			wi, err := NewGetLogsWorker()
			assert.NoError(t, err)
			w := wi.(worker)

			err = w.DoWork(config.CommonConfig{
				WorkerType: "Fake",
			}, vzlog.DefaultLogger())
			if test.doworkError == nil && test.httpDoError == nil && test.httpReadError == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.Equal(t, int64(test.successCount), w.openSearchGetSuccessCountTotal.Val)
			assert.Equal(t, int64(test.failureCount), w.openSearchGetFailureCountTotal.Val)
		})
	}
}

func (f *fakeHTTP) Do(_ *http.Request) (resp *http.Response, err error) {
	return f.resp, f.httpDoError
}

func (f fakeBody) ReadAll(d []byte) (n int, err error) {
	if f.httpReadError != nil {
		return 0, f.httpReadError
	}
	copy(d, f.bodyData)
	return len(f.bodyData), nil
}

func (f fakeBody) Read(d []byte) (n int, err error) {
	if f.httpReadError != nil {
		return 0, f.httpReadError
	}
	copy(d, f.bodyData)
	return len(f.bodyData), io.EOF
}

func (f fakeBody) Close() error {
	return nil
}
