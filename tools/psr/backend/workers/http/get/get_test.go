// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package http

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"net/http"
	"strings"
	"testing"
)

type fakeHttp struct {
	error
	resp     *http.Response
	bodyData string
}

type fakeBody struct {
	data string
}

// TestGetters tests the worker getters
// GIVEN a worker
//
//	WHEN the getter methods are calls
//	THEN ensure that the correct results are returned
func TestGetters(t *testing.T) {
	w, err := NewHTTPGetWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypeHTTPGet, wd.WorkerType)
	assert.Equal(t, "The get worker makes GET request on the given endpoint", wd.Description)
	assert.Equal(t, config.WorkerTypeHTTPGet, wd.MetricsName)

	logged := w.WantLoopInfoLogged()
	assert.False(t, logged)
}

func TestGetMetricDescList(t *testing.T) {
	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "psr_httpget_get_request_count_total", help: "The total number of GET requests"},
		{name: "2", fqName: "get_request_succeeded_count_total", help: "The total number of successful GET requests"},
		{name: "3", fqName: "get_request_failed_count_total", help: "The total number of failed GET requests"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewHTTPGetWorker()
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
		{name: "1", fqName: "psr_httpget_get_request_count_total", help: "The total number of GET requests"},
		{name: "2", fqName: "get_request_succeeded_count_total", help: "The total number of successful GET requests"},
		{name: "3", fqName: "get_request_failed_count_total", help: "The total number of failed GET requests"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewHTTPGetWorker()
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
		bodyData     string
		error        error
		statusCode   int
		reqCount     int
		successCount int
		failureCount int
	}{
		{
			name:         "1",
			bodyData:     "testsuccess",
			statusCode:   200,
			reqCount:     1,
			successCount: 1,
			failureCount: 0,
		},
		{
			name:         "1",
			bodyData:     "testerror",
			error:        errors.New("error"),
			reqCount:     1,
			successCount: 0,
			failureCount: 1,
		},
		{
			name:         "1",
			bodyData:     "testRespError",
			statusCode:   500,
			reqCount:     1,
			successCount: 0,
			failureCount: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := httpGetFunc
			defer func() {
				httpGetFunc = f
			}()
			httpGetFunc = fakeHttp{
				bodyData: test.bodyData,
				error:    test.error,
				resp: &http.Response{
					StatusCode:    test.statusCode,
					Body:          &fakeBody{data: test.bodyData},
					ContentLength: int64(len(test.bodyData)),
				},
			}.Get

			wi, err := NewHTTPGetWorker()
			w := wi.(worker)

			err = w.DoWork(config.CommonConfig{
				WorkerType: "Fake",
			}, vzlog.DefaultLogger())
			if test.error == nil {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.Equal(t, int64(test.reqCount), w.getRequestsCountTotal.Val)
			assert.Equal(t, int64(test.successCount), w.getRequestsSucceededCountTotal.Val)
			assert.Equal(t, int64(test.failureCount), w.getRequestsFailedCountTotal.Val)
		})
	}
}

func (f fakeHttp) Get(url string) (resp *http.Response, err error) {
	return f.resp, f.error
}

func (f fakeBody) Read(d []byte) (n int, err error) {
	copy(d, f.data)
	return len(f.data), nil
}

func (f fakeBody) Close() error {
	return nil
}
