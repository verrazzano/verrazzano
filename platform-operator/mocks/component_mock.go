// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
//

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi (interfaces: ComponentContext,ComponentInfo,ComponentInstaller,ComponentUpgrader,Component)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	controllerspi "github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	vzlog "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	v1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	v1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	spi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	types "k8s.io/apimachinery/pkg/types"
	reflect "reflect"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockComponentContext is a mock of ComponentContext interface
type MockComponentContext struct {
	ctrl     *gomock.Controller
	recorder *MockComponentContextMockRecorder
}

// MockComponentContextMockRecorder is the mock recorder for MockComponentContext
type MockComponentContextMockRecorder struct {
	mock *MockComponentContext
}

// NewMockComponentContext creates a new mock instance
func NewMockComponentContext(ctrl *gomock.Controller) *MockComponentContext {
	mock := &MockComponentContext{ctrl: ctrl}
	mock.recorder = &MockComponentContextMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockComponentContext) EXPECT() *MockComponentContextMockRecorder {
	return m.recorder
}

// ActualCR mocks base method
func (m *MockComponentContext) ActualCR() *v1alpha1.Verrazzano {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActualCR")
	ret0, _ := ret[0].(*v1alpha1.Verrazzano)
	return ret0
}

// ActualCR indicates an expected call of ActualCR
func (mr *MockComponentContextMockRecorder) ActualCR() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActualCR", reflect.TypeOf((*MockComponentContext)(nil).ActualCR))
}

// ActualCRV1Beta1 mocks base method
func (m *MockComponentContext) ActualCRV1Beta1() *v1beta1.Verrazzano {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActualCRV1Beta1")
	ret0, _ := ret[0].(*v1beta1.Verrazzano)
	return ret0
}

// ActualCRV1Beta1 indicates an expected call of ActualCRV1Beta1
func (mr *MockComponentContextMockRecorder) ActualCRV1Beta1() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActualCRV1Beta1", reflect.TypeOf((*MockComponentContext)(nil).ActualCRV1Beta1))
}

// Client mocks base method
func (m *MockComponentContext) Client() client.Client {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Client")
	ret0, _ := ret[0].(client.Client)
	return ret0
}

// Client indicates an expected call of Client
func (mr *MockComponentContextMockRecorder) Client() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Client", reflect.TypeOf((*MockComponentContext)(nil).Client))
}

// Copy mocks base method
func (m *MockComponentContext) Copy() spi.ComponentContext {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Copy")
	ret0, _ := ret[0].(spi.ComponentContext)
	return ret0
}

// Copy indicates an expected call of Copy
func (mr *MockComponentContextMockRecorder) Copy() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Copy", reflect.TypeOf((*MockComponentContext)(nil).Copy))
}

// EffectiveCR mocks base method
func (m *MockComponentContext) EffectiveCR() *v1alpha1.Verrazzano {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EffectiveCR")
	ret0, _ := ret[0].(*v1alpha1.Verrazzano)
	return ret0
}

// EffectiveCR indicates an expected call of EffectiveCR
func (mr *MockComponentContextMockRecorder) EffectiveCR() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EffectiveCR", reflect.TypeOf((*MockComponentContext)(nil).EffectiveCR))
}

// EffectiveCRV1Beta1 mocks base method
func (m *MockComponentContext) EffectiveCRV1Beta1() *v1beta1.Verrazzano {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EffectiveCRV1Beta1")
	ret0, _ := ret[0].(*v1beta1.Verrazzano)
	return ret0
}

// EffectiveCRV1Beta1 indicates an expected call of EffectiveCRV1Beta1
func (mr *MockComponentContextMockRecorder) EffectiveCRV1Beta1() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EffectiveCRV1Beta1", reflect.TypeOf((*MockComponentContext)(nil).EffectiveCRV1Beta1))
}

// GetComponent mocks base method
func (m *MockComponentContext) GetComponent() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetComponent")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetComponent indicates an expected call of GetComponent
func (mr *MockComponentContextMockRecorder) GetComponent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetComponent", reflect.TypeOf((*MockComponentContext)(nil).GetComponent))
}

// GetOperation mocks base method
func (m *MockComponentContext) GetOperation() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOperation")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetOperation indicates an expected call of GetOperation
func (mr *MockComponentContextMockRecorder) GetOperation() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOperation", reflect.TypeOf((*MockComponentContext)(nil).GetOperation))
}

// Init mocks base method
func (m *MockComponentContext) Init(arg0 string) spi.ComponentContext {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Init", arg0)
	ret0, _ := ret[0].(spi.ComponentContext)
	return ret0
}

// Init indicates an expected call of Init
func (mr *MockComponentContextMockRecorder) Init(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockComponentContext)(nil).Init), arg0)
}

// IsDryRun mocks base method
func (m *MockComponentContext) IsDryRun() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsDryRun")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsDryRun indicates an expected call of IsDryRun
func (mr *MockComponentContextMockRecorder) IsDryRun() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsDryRun", reflect.TypeOf((*MockComponentContext)(nil).IsDryRun))
}

// Log mocks base method
func (m *MockComponentContext) Log() vzlog.VerrazzanoLogger {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Log")
	ret0, _ := ret[0].(vzlog.VerrazzanoLogger)
	return ret0
}

// Log indicates an expected call of Log
func (mr *MockComponentContextMockRecorder) Log() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Log", reflect.TypeOf((*MockComponentContext)(nil).Log))
}

// Operation mocks base method
func (m *MockComponentContext) Operation(arg0 string) spi.ComponentContext {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Operation", arg0)
	ret0, _ := ret[0].(spi.ComponentContext)
	return ret0
}

// Operation indicates an expected call of Operation
func (mr *MockComponentContextMockRecorder) Operation(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Operation", reflect.TypeOf((*MockComponentContext)(nil).Operation), arg0)
}

// MockComponentInfo is a mock of ComponentInfo interface
type MockComponentInfo struct {
	ctrl     *gomock.Controller
	recorder *MockComponentInfoMockRecorder
}

// MockComponentInfoMockRecorder is the mock recorder for MockComponentInfo
type MockComponentInfoMockRecorder struct {
	mock *MockComponentInfo
}

// NewMockComponentInfo creates a new mock instance
func NewMockComponentInfo(ctrl *gomock.Controller) *MockComponentInfo {
	mock := &MockComponentInfo{ctrl: ctrl}
	mock.recorder = &MockComponentInfoMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockComponentInfo) EXPECT() *MockComponentInfoMockRecorder {
	return m.recorder
}

// GetCertificateNames mocks base method
func (m *MockComponentInfo) GetCertificateNames(arg0 spi.ComponentContext) []types.NamespacedName {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCertificateNames", arg0)
	ret0, _ := ret[0].([]types.NamespacedName)
	return ret0
}

// GetCertificateNames indicates an expected call of GetCertificateNames
func (mr *MockComponentInfoMockRecorder) GetCertificateNames(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCertificateNames", reflect.TypeOf((*MockComponentInfo)(nil).GetCertificateNames), arg0)
}

// GetDependencies mocks base method
func (m *MockComponentInfo) GetDependencies() []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDependencies")
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetDependencies indicates an expected call of GetDependencies
func (mr *MockComponentInfoMockRecorder) GetDependencies() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDependencies", reflect.TypeOf((*MockComponentInfo)(nil).GetDependencies))
}

// GetIngressNames mocks base method
func (m *MockComponentInfo) GetIngressNames(arg0 spi.ComponentContext) []types.NamespacedName {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngressNames", arg0)
	ret0, _ := ret[0].([]types.NamespacedName)
	return ret0
}

// GetIngressNames indicates an expected call of GetIngressNames
func (mr *MockComponentInfoMockRecorder) GetIngressNames(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngressNames", reflect.TypeOf((*MockComponentInfo)(nil).GetIngressNames), arg0)
}

// GetJSONName mocks base method
func (m *MockComponentInfo) GetJSONName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetJSONName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetJSONName indicates an expected call of GetJSONName
func (mr *MockComponentInfoMockRecorder) GetJSONName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetJSONName", reflect.TypeOf((*MockComponentInfo)(nil).GetJSONName))
}

// GetMinVerrazzanoVersion mocks base method
func (m *MockComponentInfo) GetMinVerrazzanoVersion() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMinVerrazzanoVersion")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetMinVerrazzanoVersion indicates an expected call of GetMinVerrazzanoVersion
func (mr *MockComponentInfoMockRecorder) GetMinVerrazzanoVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMinVerrazzanoVersion", reflect.TypeOf((*MockComponentInfo)(nil).GetMinVerrazzanoVersion))
}

// GetOverrides mocks base method
func (m *MockComponentInfo) GetOverrides(arg0 runtime.Object) interface{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOverrides", arg0)
	ret0, _ := ret[0].(interface{})
	return ret0
}

// GetOverrides indicates an expected call of GetOverrides
func (mr *MockComponentInfoMockRecorder) GetOverrides(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOverrides", reflect.TypeOf((*MockComponentInfo)(nil).GetOverrides), arg0)
}

// IsAvailable mocks base method
func (m *MockComponentInfo) IsAvailable(arg0 spi.ComponentContext) (string, v1alpha1.ComponentAvailability) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsAvailable", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(v1alpha1.ComponentAvailability)
	return ret0, ret1
}

// IsAvailable indicates an expected call of IsAvailable
func (mr *MockComponentInfoMockRecorder) IsAvailable(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsAvailable", reflect.TypeOf((*MockComponentInfo)(nil).IsAvailable), arg0)
}

// IsEnabled mocks base method
func (m *MockComponentInfo) IsEnabled(arg0 runtime.Object) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsEnabled", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsEnabled indicates an expected call of IsEnabled
func (mr *MockComponentInfoMockRecorder) IsEnabled(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsEnabled", reflect.TypeOf((*MockComponentInfo)(nil).IsEnabled), arg0)
}

// IsReady mocks base method
func (m *MockComponentInfo) IsReady(arg0 spi.ComponentContext) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsReady", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsReady indicates an expected call of IsReady
func (mr *MockComponentInfoMockRecorder) IsReady(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsReady", reflect.TypeOf((*MockComponentInfo)(nil).IsReady), arg0)
}

// MonitorOverrides mocks base method
func (m *MockComponentInfo) MonitorOverrides(arg0 spi.ComponentContext) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MonitorOverrides", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// MonitorOverrides indicates an expected call of MonitorOverrides
func (mr *MockComponentInfoMockRecorder) MonitorOverrides(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MonitorOverrides", reflect.TypeOf((*MockComponentInfo)(nil).MonitorOverrides), arg0)
}

// Name mocks base method
func (m *MockComponentInfo) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (mr *MockComponentInfoMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockComponentInfo)(nil).Name))
}

// Namespace mocks base method
func (m *MockComponentInfo) Namespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Namespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// Namespace indicates an expected call of Namespace
func (mr *MockComponentInfoMockRecorder) Namespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Namespace", reflect.TypeOf((*MockComponentInfo)(nil).Namespace))
}

// ShouldInstallBeforeUpgrade mocks base method
func (m *MockComponentInfo) ShouldInstallBeforeUpgrade() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ShouldInstallBeforeUpgrade")
	ret0, _ := ret[0].(bool)
	return ret0
}

// ShouldInstallBeforeUpgrade indicates an expected call of ShouldInstallBeforeUpgrade
func (mr *MockComponentInfoMockRecorder) ShouldInstallBeforeUpgrade() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShouldInstallBeforeUpgrade", reflect.TypeOf((*MockComponentInfo)(nil).ShouldInstallBeforeUpgrade))
}

// MockComponentInstaller is a mock of ComponentInstaller interface
type MockComponentInstaller struct {
	ctrl     *gomock.Controller
	recorder *MockComponentInstallerMockRecorder
}

// MockComponentInstallerMockRecorder is the mock recorder for MockComponentInstaller
type MockComponentInstallerMockRecorder struct {
	mock *MockComponentInstaller
}

// NewMockComponentInstaller creates a new mock instance
func NewMockComponentInstaller(ctrl *gomock.Controller) *MockComponentInstaller {
	mock := &MockComponentInstaller{ctrl: ctrl}
	mock.recorder = &MockComponentInstallerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockComponentInstaller) EXPECT() *MockComponentInstallerMockRecorder {
	return m.recorder
}

// Install mocks base method
func (m *MockComponentInstaller) Install(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Install", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Install indicates an expected call of Install
func (mr *MockComponentInstallerMockRecorder) Install(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Install", reflect.TypeOf((*MockComponentInstaller)(nil).Install), arg0)
}

// IsInstalled mocks base method
func (m *MockComponentInstaller) IsInstalled(arg0 spi.ComponentContext) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsInstalled", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsInstalled indicates an expected call of IsInstalled
func (mr *MockComponentInstallerMockRecorder) IsInstalled(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsInstalled", reflect.TypeOf((*MockComponentInstaller)(nil).IsInstalled), arg0)
}

// IsOperatorInstallSupported mocks base method
func (m *MockComponentInstaller) IsOperatorInstallSupported() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsOperatorInstallSupported")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsOperatorInstallSupported indicates an expected call of IsOperatorInstallSupported
func (mr *MockComponentInstallerMockRecorder) IsOperatorInstallSupported() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsOperatorInstallSupported", reflect.TypeOf((*MockComponentInstaller)(nil).IsOperatorInstallSupported))
}

// PostInstall mocks base method
func (m *MockComponentInstaller) PostInstall(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostInstall", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostInstall indicates an expected call of PostInstall
func (mr *MockComponentInstallerMockRecorder) PostInstall(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostInstall", reflect.TypeOf((*MockComponentInstaller)(nil).PostInstall), arg0)
}

// PreInstall mocks base method
func (m *MockComponentInstaller) PreInstall(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreInstall", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreInstall indicates an expected call of PreInstall
func (mr *MockComponentInstallerMockRecorder) PreInstall(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreInstall", reflect.TypeOf((*MockComponentInstaller)(nil).PreInstall), arg0)
}

// MockComponentUpgrader is a mock of ComponentUpgrader interface
type MockComponentUpgrader struct {
	ctrl     *gomock.Controller
	recorder *MockComponentUpgraderMockRecorder
}

// MockComponentUpgraderMockRecorder is the mock recorder for MockComponentUpgrader
type MockComponentUpgraderMockRecorder struct {
	mock *MockComponentUpgrader
}

// NewMockComponentUpgrader creates a new mock instance
func NewMockComponentUpgrader(ctrl *gomock.Controller) *MockComponentUpgrader {
	mock := &MockComponentUpgrader{ctrl: ctrl}
	mock.recorder = &MockComponentUpgraderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockComponentUpgrader) EXPECT() *MockComponentUpgraderMockRecorder {
	return m.recorder
}

// PostUpgrade mocks base method
func (m *MockComponentUpgrader) PostUpgrade(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostUpgrade", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostUpgrade indicates an expected call of PostUpgrade
func (mr *MockComponentUpgraderMockRecorder) PostUpgrade(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostUpgrade", reflect.TypeOf((*MockComponentUpgrader)(nil).PostUpgrade), arg0)
}

// PreUpgrade mocks base method
func (m *MockComponentUpgrader) PreUpgrade(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreUpgrade", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreUpgrade indicates an expected call of PreUpgrade
func (mr *MockComponentUpgraderMockRecorder) PreUpgrade(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreUpgrade", reflect.TypeOf((*MockComponentUpgrader)(nil).PreUpgrade), arg0)
}

// Upgrade mocks base method
func (m *MockComponentUpgrader) Upgrade(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Upgrade", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Upgrade indicates an expected call of Upgrade
func (mr *MockComponentUpgraderMockRecorder) Upgrade(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upgrade", reflect.TypeOf((*MockComponentUpgrader)(nil).Upgrade), arg0)
}

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

// GetCertificateNames mocks base method
func (m *MockComponent) GetCertificateNames(arg0 spi.ComponentContext) []types.NamespacedName {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCertificateNames", arg0)
	ret0, _ := ret[0].([]types.NamespacedName)
	return ret0
}

// GetCertificateNames indicates an expected call of GetCertificateNames
func (mr *MockComponentMockRecorder) GetCertificateNames(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCertificateNames", reflect.TypeOf((*MockComponent)(nil).GetCertificateNames), arg0)
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

// GetIngressNames mocks base method
func (m *MockComponent) GetIngressNames(arg0 spi.ComponentContext) []types.NamespacedName {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIngressNames", arg0)
	ret0, _ := ret[0].([]types.NamespacedName)
	return ret0
}

// GetIngressNames indicates an expected call of GetIngressNames
func (mr *MockComponentMockRecorder) GetIngressNames(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIngressNames", reflect.TypeOf((*MockComponent)(nil).GetIngressNames), arg0)
}

// GetJSONName mocks base method
func (m *MockComponent) GetJSONName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetJSONName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetJSONName indicates an expected call of GetJSONName
func (mr *MockComponentMockRecorder) GetJSONName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetJSONName", reflect.TypeOf((*MockComponent)(nil).GetJSONName))
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

// GetModuleConfigAsHelmValues mocks base method
func (m *MockComponent) GetModuleConfigAsHelmValues(arg0 *v1alpha1.Verrazzano) (*v1.JSON, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModuleConfigAsHelmValues", arg0)
	ret0, _ := ret[0].(*v1.JSON)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModuleConfigAsHelmValues indicates an expected call of GetModuleConfigAsHelmValues
func (mr *MockComponentMockRecorder) GetModuleConfigAsHelmValues(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModuleConfigAsHelmValues", reflect.TypeOf((*MockComponent)(nil).GetModuleConfigAsHelmValues), arg0)
}

// GetOverrides mocks base method
func (m *MockComponent) GetOverrides(arg0 runtime.Object) interface{} {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOverrides", arg0)
	ret0, _ := ret[0].(interface{})
	return ret0
}

// GetOverrides indicates an expected call of GetOverrides
func (mr *MockComponentMockRecorder) GetOverrides(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOverrides", reflect.TypeOf((*MockComponent)(nil).GetOverrides), arg0)
}

// GetWatchDescriptors mocks base method
func (m *MockComponent) GetWatchDescriptors() []controllerspi.WatchDescriptor {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWatchDescriptors")
	ret0, _ := ret[0].([]controllerspi.WatchDescriptor)
	return ret0
}

// GetWatchDescriptors indicates an expected call of GetWatchDescriptors
func (mr *MockComponentMockRecorder) GetWatchDescriptors() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWatchDescriptors", reflect.TypeOf((*MockComponent)(nil).GetWatchDescriptors))
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

// IsAvailable mocks base method
func (m *MockComponent) IsAvailable(arg0 spi.ComponentContext) (string, v1alpha1.ComponentAvailability) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsAvailable", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(v1alpha1.ComponentAvailability)
	return ret0, ret1
}

// IsAvailable indicates an expected call of IsAvailable
func (mr *MockComponentMockRecorder) IsAvailable(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsAvailable", reflect.TypeOf((*MockComponent)(nil).IsAvailable), arg0)
}

// IsEnabled mocks base method
func (m *MockComponent) IsEnabled(arg0 runtime.Object) bool {
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

// IsOperatorUninstallSupported mocks base method
func (m *MockComponent) IsOperatorUninstallSupported() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsOperatorUninstallSupported")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsOperatorUninstallSupported indicates an expected call of IsOperatorUninstallSupported
func (mr *MockComponentMockRecorder) IsOperatorUninstallSupported() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsOperatorUninstallSupported", reflect.TypeOf((*MockComponent)(nil).IsOperatorUninstallSupported))
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

// MonitorOverrides mocks base method
func (m *MockComponent) MonitorOverrides(arg0 spi.ComponentContext) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MonitorOverrides", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// MonitorOverrides indicates an expected call of MonitorOverrides
func (mr *MockComponentMockRecorder) MonitorOverrides(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MonitorOverrides", reflect.TypeOf((*MockComponent)(nil).MonitorOverrides), arg0)
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

// Namespace mocks base method
func (m *MockComponent) Namespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Namespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// Namespace indicates an expected call of Namespace
func (mr *MockComponentMockRecorder) Namespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Namespace", reflect.TypeOf((*MockComponent)(nil).Namespace))
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

// PostUninstall mocks base method
func (m *MockComponent) PostUninstall(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PostUninstall", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PostUninstall indicates an expected call of PostUninstall
func (mr *MockComponentMockRecorder) PostUninstall(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PostUninstall", reflect.TypeOf((*MockComponent)(nil).PostUninstall), arg0)
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

// PreUninstall mocks base method
func (m *MockComponent) PreUninstall(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PreUninstall", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// PreUninstall indicates an expected call of PreUninstall
func (mr *MockComponentMockRecorder) PreUninstall(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PreUninstall", reflect.TypeOf((*MockComponent)(nil).PreUninstall), arg0)
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

// Reconcile mocks base method
func (m *MockComponent) Reconcile(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reconcile", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Reconcile indicates an expected call of Reconcile
func (mr *MockComponentMockRecorder) Reconcile(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reconcile", reflect.TypeOf((*MockComponent)(nil).Reconcile), arg0)
}

// ShouldInstallBeforeUpgrade mocks base method
func (m *MockComponent) ShouldInstallBeforeUpgrade() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ShouldInstallBeforeUpgrade")
	ret0, _ := ret[0].(bool)
	return ret0
}

// ShouldInstallBeforeUpgrade indicates an expected call of ShouldInstallBeforeUpgrade
func (mr *MockComponentMockRecorder) ShouldInstallBeforeUpgrade() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShouldInstallBeforeUpgrade", reflect.TypeOf((*MockComponent)(nil).ShouldInstallBeforeUpgrade))
}

// ShouldUseModule mocks base method
func (m *MockComponent) ShouldUseModule() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ShouldUseModule")
	ret0, _ := ret[0].(bool)
	return ret0
}

// ShouldUseModule indicates an expected call of ShouldUseModule
func (mr *MockComponentMockRecorder) ShouldUseModule() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShouldUseModule", reflect.TypeOf((*MockComponent)(nil).ShouldUseModule))
}

// Uninstall mocks base method
func (m *MockComponent) Uninstall(arg0 spi.ComponentContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Uninstall", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Uninstall indicates an expected call of Uninstall
func (mr *MockComponentMockRecorder) Uninstall(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Uninstall", reflect.TypeOf((*MockComponent)(nil).Uninstall), arg0)
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

// ValidateInstall mocks base method
func (m *MockComponent) ValidateInstall(arg0 *v1alpha1.Verrazzano) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateInstall", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidateInstall indicates an expected call of ValidateInstall
func (mr *MockComponentMockRecorder) ValidateInstall(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateInstall", reflect.TypeOf((*MockComponent)(nil).ValidateInstall), arg0)
}

// ValidateInstallV1Beta1 mocks base method
func (m *MockComponent) ValidateInstallV1Beta1(arg0 *v1beta1.Verrazzano) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateInstallV1Beta1", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidateInstallV1Beta1 indicates an expected call of ValidateInstallV1Beta1
func (mr *MockComponentMockRecorder) ValidateInstallV1Beta1(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateInstallV1Beta1", reflect.TypeOf((*MockComponent)(nil).ValidateInstallV1Beta1), arg0)
}

// ValidateUpdate mocks base method
func (m *MockComponent) ValidateUpdate(arg0, arg1 *v1alpha1.Verrazzano) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateUpdate", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidateUpdate indicates an expected call of ValidateUpdate
func (mr *MockComponentMockRecorder) ValidateUpdate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateUpdate", reflect.TypeOf((*MockComponent)(nil).ValidateUpdate), arg0, arg1)
}

// ValidateUpdateV1Beta1 mocks base method
func (m *MockComponent) ValidateUpdateV1Beta1(arg0, arg1 *v1beta1.Verrazzano) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateUpdateV1Beta1", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidateUpdateV1Beta1 indicates an expected call of ValidateUpdateV1Beta1
func (mr *MockComponentMockRecorder) ValidateUpdateV1Beta1(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateUpdateV1Beta1", reflect.TypeOf((*MockComponent)(nil).ValidateUpdateV1Beta1), arg0, arg1)
}
