// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi (interfaces: Component)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	spi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	reflect "reflect"
)

// MockComponent is a mock of Component interface
type MockComponent struct {
	ctrl     *gomock.Controller
	recorder *MockComponentMockRecorder
}

// MockComponentMockRecorder is the mock recorder for MockComponent
type MockComponentMockRecorder struct {
	mock *MockComponent
}

// NewMockComponent creates a new mock instance
func NewMockComponent(ctrl *gomock.Controller) *MockComponent {
	mock := &MockComponent{ctrl: ctrl}
	mock.recorder = &MockComponentMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockComponent) EXPECT() *MockComponentMockRecorder {
	return m.recorder
}

// GetDependencies mocks base method
func (m *MockComponent) GetDependencies() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDependencies")
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetDependencies indicates an expected call of GetDependencies
func (mr *MockComponentMockRecorder) GetDependencies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDependencies", reflect.TypeOf((*MockComponent)(nil).GetDependencies))
}

// GetMinVerrazzanoVersion mocks base method
func (m *MockComponent) GetMinVerrazzanoVersion() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMinVerrazzanoVersion")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetMinVerrazzanoVersion indicates an expected call of GetMinVerrazzanoVersion
func (mr *MockComponentMockRecorder) GetMinVerrazzanoVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMinVerrazzanoVersion", reflect.TypeOf((*MockComponent)(nil).GetMinVerrazzanoVersion))
}

// Install mocks base method
func (m *MockComponent) Install(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Install", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Install indicates an expected call of Install
func (mr *MockComponentMockRecorder) Install(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Install", reflect.TypeOf((*MockComponent)(nil).Install), arg0)
}

// IsEnabled mocks base method
func (m *MockComponent) IsEnabled(arg0 spi.ComponentContext) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsEnabled", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsEnabled indicates an expected call of IsEnabled
func (mr *MockComponentMockRecorder) IsEnabled(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsEnabled", reflect.TypeOf((*MockComponent)(nil).IsEnabled), arg0)
}

// IsInstalled mocks base method
func (m *MockComponent) IsInstalled(arg0 spi.ComponentContext) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsInstalled", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsInstalled indicates an expected call of IsInstalled
func (mr *MockComponentMockRecorder) IsInstalled(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsInstalled", reflect.TypeOf((*MockComponent)(nil).IsInstalled), arg0)
}

// IsOperatorInstallSupported mocks base method
func (m *MockComponent) IsOperatorInstallSupported() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsOperatorInstallSupported")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsOperatorInstallSupported indicates an expected call of IsOperatorInstallSupported
func (mr *MockComponentMockRecorder) IsOperatorInstallSupported() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsOperatorInstallSupported", reflect.TypeOf((*MockComponent)(nil).IsOperatorInstallSupported))
}

// IsReady mocks base method
func (m *MockComponent) IsReady(arg0 spi.ComponentContext) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsReady", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsReady indicates an expected call of IsReady
func (mr *MockComponentMockRecorder) IsReady(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsReady", reflect.TypeOf((*MockComponent)(nil).IsReady), arg0)
}

// Name mocks base method
func (m *MockComponent) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (mr *MockComponentMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockComponent)(nil).Name))
}

// PostInstall mocks base method
func (m *MockComponent) PostInstall(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostInstall", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostInstall indicates an expected call of PostInstall
func (mr *MockComponentMockRecorder) PostInstall(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostInstall", reflect.TypeOf((*MockComponent)(nil).PostInstall), arg0)
}

// PostUpgrade mocks base method
func (m *MockComponent) PostUpgrade(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostUpgrade", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostUpgrade indicates an expected call of PostUpgrade
func (mr *MockComponentMockRecorder) PostUpgrade(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostUpgrade", reflect.TypeOf((*MockComponent)(nil).PostUpgrade), arg0)
}

// PreInstall mocks base method
func (m *MockComponent) PreInstall(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreInstall", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreInstall indicates an expected call of PreInstall
func (mr *MockComponentMockRecorder) PreInstall(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreInstall", reflect.TypeOf((*MockComponent)(nil).PreInstall), arg0)
}

// PreUpgrade mocks base method
func (m *MockComponent) PreUpgrade(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreUpgrade", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreUpgrade indicates an expected call of PreUpgrade
func (mr *MockComponentMockRecorder) PreUpgrade(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreUpgrade", reflect.TypeOf((*MockComponent)(nil).PreUpgrade), arg0)
}

// Upgrade mocks base method
func (m *MockComponent) Upgrade(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Upgrade", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Upgrade indicates an expected call of Upgrade
func (mr *MockComponentMockRecorder) Upgrade(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upgrade", reflect.TypeOf((*MockComponent)(nil).Upgrade), arg0)
}
