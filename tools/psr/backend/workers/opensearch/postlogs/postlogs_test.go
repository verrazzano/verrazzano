// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package postlogs

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

type fakeEnv struct {
	data map[string]string
}

type fakeHttp struct {
	resp        *http.Response
	httpDoError error
}

type fakeBody struct {
	bodyData      string
	httpReadError error
}

var _ httpClientI = &fakeHttp{}

// TestGetters tests the worker getters
// GIVEN a worker
//
//	WHEN the getter methods are calls
//	THEN ensure that the correct results are returned
func TestGetters(t *testing.T) {
	w, err := NewPostLogsWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypePostLogs, wd.WorkerType)
	assert.Equal(t, "The postlogs worker performs POST requests on the OpenSearch endpoint", wd.Description)
	assert.Equal(t, config.WorkerTypePostLogs, wd.MetricsName)

	logged := w.WantLoopInfoLogged()
	assert.False(t, logged)
}

// TestGetEnvDescList tests the GetEnvDescList method
// GIVEN a worker
//
//	WHEN the GetEnvDescList methods is called
//	THEN ensure that the correct results are returned
func TestGetEnvDescList(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		defval   string
		required bool
	}{
		{name: "1",
			key:      LogEntries,
			defval:   "1",
			required: false,
		},
		{name: "2",
			key:      LogLength,
			defval:   "1",
			required: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, err := NewPostLogsWorker()
			assert.NoError(t, err)
			el := w.GetEnvDescList()
			for _, e := range el {
				if e.Key == test.key {
					assert.Equal(t, test.defval, e.DefaultVal)
					assert.Equal(t, test.required, e.Required)
				}
			}
		})
	}
}

func TestGetMetricDescList(t *testing.T) {
	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "opensearch_post_success_count_total", help: "The total number of successful OpenSearch POST requests"},
		{name: "2", fqName: "opensearch_post_failure_count_total", help: "The total number of successful OpenSearch POST requests"},
		{name: "3", fqName: "opensearch_post_success_latency_nanoseconds", help: "The latency of successful OpenSearch POST requests in nanoseconds"},
		{name: "4", fqName: "opensearch_post_failure_latency_nanoseconds", help: "The latency of failed OpenSearch POST requests in nanoseconds"},
		{name: "5", fqName: "opensearch_post_data_chars_total", help: "The total number of characters posted to OpenSearch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewPostLogsWorker()
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
		{name: "1", fqName: "opensearch_post_success_count_total", help: "The total number of successful OpenSearch POST requests"},
		{name: "2", fqName: "opensearch_post_failure_count_total", help: "The total number of successful OpenSearch POST requests"},
		{name: "3", fqName: "opensearch_post_success_latency_nanoseconds", help: "The latency of successful OpenSearch POST requests in nanoseconds"},
		{name: "4", fqName: "opensearch_post_failure_latency_nanoseconds", help: "The latency of failed OpenSearch POST requests in nanoseconds"},
		{name: "5", fqName: "opensearch_post_data_chars_total", help: "The total number of characters posted to OpenSearch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewPostLogsWorker()
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
			statusCode:   201,
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
			var resp *http.Response
			if !test.nilResp {
				resp = &http.Response{
					StatusCode:    test.statusCode,
					Body:          &test.fakeBody,
					ContentLength: int64(len(test.bodyData)),
				}
			}

			c := httpClient
			defer func() {
				httpClient = c
			}()
			httpClient = &fakeHttp{
				httpDoError: test.doworkError,
				resp:        resp,
			}

			wi, err := NewPostLogsWorker()
			w := wi.(worker)

			err = config.PsrEnv.LoadFromEnv(w.GetEnvDescList())
			assert.NoError(t, err)

			err = w.DoWork(config.CommonConfig{
				WorkerType: "Fake",
			}, vzlog.DefaultLogger())
			if test.doworkError == nil && test.httpDoError == nil && test.httpReadError == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.Equal(t, int64(test.successCount), w.openSearchPostSuccessCountTotal.Val)
			assert.Equal(t, int64(test.failureCount), w.openSearchPostFailureCountTotal.Val)
		})
	}
}

func (f *fakeHttp) Do(_ *http.Request) (resp *http.Response, err error) {
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

func (f *fakeEnv) GetEnv(key string) string {
	return f.data[key]
}
