// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by MockGen. DO NOT EDIT.
// Source: platform-operator/controllers/clusters/rancher.go

// Package mocks is a generated GoMock package.
package mocks

import (
	http "net/http"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockrequestSender is a mock of requestSender interface.
type MockRequestSender struct {
	ctrl     *gomock.Controller
	recorder *MockrequestSenderMockRecorder
}

// MockrequestSenderMockRecorder is the mock recorder for MockrequestSender.
type MockrequestSenderMockRecorder struct {
	mock *MockRequestSender
}

// NewMockrequestSender creates a new mock instance.
func NewMockRequestSender(ctrl *gomock.Controller) *MockRequestSender {
	mock := &MockRequestSender{ctrl: ctrl}
	mock.recorder = &MockrequestSenderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRequestSender) EXPECT() *MockrequestSenderMockRecorder {
	return m.recorder
}

// Do mocks base method.
func (m *MockRequestSender) Do(httpClient *http.Client, req *http.Request) (*http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Do", httpClient, req)
	ret0, _ := ret[0].(*http.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Do indicates an expected call of Do.
func (mr *MockrequestSenderMockRecorder) Do(httpClient, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Do", reflect.TypeOf((*MockRequestSender)(nil).Do), httpClient, req)
}
