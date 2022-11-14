// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scale

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"net/http"
	"strings"
	"testing"
)

type fakeHTTP struct {
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
	w, err := NewScaleWorker()
	assert.NoError(t, err)

	wd := w.GetWorkerDesc()
	assert.Equal(t, config.WorkerTypeScale, wd.WorkerType)
	assert.Equal(t, "The OpenSearch scale worker scales an OpenSearch tier in and out continuously", wd.Description)
	assert.Equal(t, config.WorkerTypeScale, wd.MetricsName)

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
			key:      openSearchTier,
			defval:   "",
			required: true,
		},
		{name: "2",
			key:      minReplicaCount,
			defval:   "3",
			required: false,
		},
		{name: "3",
			key:      maxReplicaCount,
			defval:   "5",
			required: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, err := NewScaleWorker()
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

// TestGetMetricDescList tests the GetEnvDescList method
// GIVEN a worker
//
//	WHEN the GetEnvDescList methods is called
//	THEN ensure that the correct results are returned
func TestGetMetricDescList(t *testing.T) {
	tests := []struct {
		name   string
		fqName string
		help   string
	}{
		{name: "1", fqName: "opensearch_scale_out_count_total", help: "The total number of times OpenSearch scaled out"},
		{name: "2", fqName: "opensearch_scale_in_count_total", help: "The total number of times OpenSearch scaled in"},
		{name: "3", fqName: "opensearch_scale_out_seconds", help: "The number of seconds elapsed to scale out OpenSearch"},
		{name: "4", fqName: "opensearch_scale_in_seconds", help: "The number of seconds elapsed to scale in OpenSearch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewScaleWorker()
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
		{name: "1", fqName: "opensearch_scale_out_count_total", help: "The total number of times OpenSearch scaled out"},
		{name: "2", fqName: "opensearch_scale_in_count_total", help: "The total number of times OpenSearch scaled in"},
		{name: "3", fqName: "opensearch_scale_out_seconds", help: "The number of seconds elapsed to scale out OpenSearch"},
		{name: "4", fqName: "opensearch_scale_in_seconds", help: "The number of seconds elapsed to scale in OpenSearch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wi, err := NewScaleWorker()
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

//// TestDoWork tests the DoWork method
//// GIVEN a worker
////
////	WHEN the DoWork methods is called
////	THEN ensure that the correct results are returned
//func TestDoWork(t *testing.T) {
//	tests := []struct {
//		name         string
//		bodyData     string
//		getError     error
//		doworkError  error
//		statusCode   int
//		nilResp      bool
//		reqCount     int
//		successCount int
//		failureCount int
//	}{
//		{
//			name:         "1",
//			bodyData:     "testsuccess",
//			statusCode:   200,
//			reqCount:     1,
//			successCount: 1,
//			failureCount: 0,
//		},
//		{
//			name:         "2",
//			bodyData:     "testerror",
//			getError:     errors.New("error"),
//			reqCount:     1,
//			successCount: 0,
//			failureCount: 1,
//		},
//		{
//			name:         "3",
//			bodyData:     "testRespError",
//			statusCode:   500,
//			reqCount:     1,
//			successCount: 0,
//			failureCount: 1,
//		},
//		{
//			name:         "4",
//			bodyData:     "testNilResp",
//			doworkError:  errors.New("GET request to endpoint received a nil response"),
//			nilResp:      true,
//			reqCount:     1,
//			successCount: 0,
//			failureCount: 1,
//		},
//	}
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			f := httpGetFunc
//			defer func() {
//				httpGetFunc = f
//			}()
//			var resp *http.Response
//			if !test.nilResp {
//				resp = &http.Response{
//					StatusCode:    test.statusCode,
//					Body:          &fakeBody{data: test.bodyData},
//					ContentLength: int64(len(test.bodyData)),
//				}
//			}
//			httpGetFunc = fakeHTTP{
//				bodyData: test.bodyData,
//				error:    test.getError,
//				resp:     resp,
//			}.Get
//
//			wi, err := NewScaleWorker()
//			assert.NoError(t, err)
//			w := wi.(worker)
//			err = w.DoWork(config.CommonConfig{
//				WorkerType: "Fake",
//			}, vzlog.DefaultLogger())
//			if test.doworkError == nil && test.getError == nil {
//				assert.NoError(t, err)
//			} else {
//				assert.Error(t, err)
//			}
//
//			assert.Equal(t, int64(test.reqCount), w.getRequestsCountTotal.Val)
//			assert.Equal(t, int64(test.successCount), w.getRequestsSucceededCountTotal.Val)
//			assert.Equal(t, int64(test.failureCount), w.getRequestsFailedCountTotal.Val)
//		})
//	}
//}
//
//func (f fakeHTTP) Get(url string) (resp *http.Response, err error) {
//	return f.resp, f.error
//}
//
//func (f fakeBody) Read(d []byte) (n int, err error) {
//	copy(d, f.data)
//	return len(f.data), nil
//}
//
//func (f fakeBody) Close() error {
//	return nil
//}
