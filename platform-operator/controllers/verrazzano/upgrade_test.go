// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamapi "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	helm2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/rbac"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vzinstance"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	gofake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// unitTestBomFIle is used for unit test
const unitTestBomFile = "../../verrazzano-bom.json"

// ingress list constants
const dnsDomain = "myenv.testverrazzano.com"
const keycloakURL = "keycloak." + dnsDomain
const esURL = "elasticsearch." + dnsDomain
const promURL = "prometheus." + dnsDomain
const grafanaURL = "grafana." + dnsDomain
const kialiURL = "kiali." + dnsDomain
const kibanaURL = "kibana." + dnsDomain
const rancherURL = "rancher." + dnsDomain
const consoleURL = "verrazzano." + dnsDomain
const jaegerURL = "jaeger." + dnsDomain

var istioEnabled = false
var jaegerEnabled = true

// TestUpgradeNoVersion tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile a verrazzano resource after install is completed
// WHEN a verrazzano version is empty
// THEN ensure a condition with type UpgradeStarted is not added
func TestUpgradeNoVersion(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	keycloakEnabled := false
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Keycloak: &vzapi.KeycloakComponent{
						Enabled: &keycloakEnabled,
					},
					Istio: &vzapi.IstioComponent{
						Enabled: &istioEnabled,
					},
				},
			},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
				Components: func() vzapi.ComponentStatusMap {
					statusMap := makeVerrazzanoComponentStatusMap()
					statusMap[keycloak.ComponentName].State = vzapi.CompStateDisabled
					statusMap[istio.ComponentName].State = vzapi.CompStateDisabled
					return statusMap
				}(),
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, labels),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stub out the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")

	// check for upgrade started condition not true
	found := false
	for _, condition := range verrazzano.Status.Conditions {
		if condition.Type == vzapi.CondUpgradeStarted {
			found = true
			break
		}
	}
	asserts.False(found, "expected upgrade started to be false")
}

// TestUpgradeSameVersion tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN a verrazzano spec.version is the same as the status.version
// THEN ensure a condition with type UpgradeStarted is not added
func TestUpgradeSameVersion(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	keycloakEnabled := false
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Version: "1.2.0",
				Components: vzapi.ComponentSpec{
					Keycloak: &vzapi.KeycloakComponent{
						Enabled: &keycloakEnabled,
					},
					Istio: &vzapi.IstioComponent{
						Enabled: &istioEnabled,
					},
				},
			},
			Status: vzapi.VerrazzanoStatus{
				State:   vzapi.VzStateReady,
				Version: "1.2.0",
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
				Components: func() vzapi.ComponentStatusMap {
					statusMap := makeVerrazzanoComponentStatusMap()
					statusMap[keycloak.ComponentName].State = vzapi.CompStateDisabled
					statusMap[istio.ComponentName].State = vzapi.CompStateDisabled
					return statusMap
				}(),
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, labels),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")

	// check for upgrade started condition not true
	found := false
	for _, condition := range verrazzano.Status.Conditions {
		if condition.Type == vzapi.CondUpgradeStarted {
			found = true
			break
		}
	}
	asserts.False(found, "expected upgrade started to be false")
}

// TestUpgradeInitComponents tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile a verrazzano resource when Status.Components are empty
// WHEN spec.version doesn't match status.version
// THEN ensure that the Status.components is populated
func TestUpgradeInitComponents(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Version: "1.1.0"},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
}

// TestUpgradeStarted tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile a verrazzano resource after install is completed
// WHEN upgrade has not been started and spec.version doesn't match status.version
// THEN ensure status state is updated to upgrading
func TestUpgradeStarted(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Version: "0.2.0"},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
				Components: makeVerrazzanoComponentStatusMap(),
				Version:    "0.1.0",
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.True(result.RequeueAfter.Seconds() <= 3)
	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	asserts.Equal(verrazzano.Status.State, vzapi.VzStateUpgrading)
}

// TestDeleteDuringUpgrade tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile a verrazzano resource after upgrade is started
// WHEN upgrade has started and deletion timestamp is not zero
// THEN ensure a condition with type Uninstall Started is added
func TestDeleteDuringUpgrade(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: func() metav1.ObjectMeta {
				om := createObjectMeta(namespace, name, []string{finalizerName})
				om.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				return om
			}(),
			Spec: vzapi.VerrazzanoSpec{
				Version: "0.2.0"},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStateUpgrading,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
				Components: makeVerrazzanoComponentStatusMap()},
		},
	).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.False(result.Requeue)
	asserts.Equal(time.Duration(0)*time.Second, result.RequeueAfter)

	// check for uninstall started condition
	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.Error(err)
	asserts.True(errors2.IsNotFound(err))
}

// TestUpgradeStartedWhenPrevFailures tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN the total upgrade failures exceed the limit, but the current upgrade is under the limit
// THEN ensure that upgrade is started
func TestUpgradeStartedWhenPrevFailures(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: func() metav1.ObjectMeta {
				om := createObjectMeta(namespace, name, []string{finalizerName})
				om.Generation = 2
				return om
			}(),
			Spec: vzapi.VerrazzanoSpec{
				Version: "0.2.0"},
			Status: vzapi.VerrazzanoStatus{
				Version:    "0.1.0",
				State:      vzapi.VzStateReady,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type:    vzapi.CondUpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.CondUpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type:    vzapi.CondUpgradeFailed,
						Message: "Upgrade failed generation:1",
					},
					{
						Type: vzapi.CondUpgradeComplete,
					},
					{
						Type:    vzapi.CondUpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
					{
						Type:    vzapi.CondUpgradeFailed,
						Message: "Upgrade failed generation:2",
					},
				},
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.True(result.RequeueAfter.Seconds() <= 3)
	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	asserts.Equal(verrazzano.Status.State, vzapi.VzStateUpgrading)
}

// TestUpgradeCompleted tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile a verrazzano resource after install is completed
// WHEN spec.version doesn't match status.version
// THEN ensure a condition with type UpgradeCompleted is added
func TestUpgradeCompleted(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	fname, _ := filepath.Abs(unitTestBomFile)
	config.SetDefaultBomFilePath(fname)
	asserts := assert.New(t)

	// Setup fake client to provide workloads for restart platform testing
	goClient, err := initFakeClient()
	asserts.NoError(err)
	k8sutil.SetFakeClient(goClient)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{},
		}
	})
	defer registry.ResetGetComponentsFn()

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	_ = oamcore.AddToScheme(k8scheme.Scheme)

	authConfig := createKeycloakAuthConfig()
	localAuthConfig := createLocalAuthConfig()
	kcSecret := createKeycloakSecret()
	firstLoginSetting := createFirstLoginSetting()
	rancherIngress := createIngress(common.CattleSystem, constants.RancherIngress, common.RancherName)
	kcIngress := createIngress(constants.KeycloakNamespace, constants.KeycloakIngress, constants.KeycloakIngress)
	verrazzanoAdminClusterRole := createClusterRoles("verrazzano-admin")
	verrazzanoMonitorClusterRole := createClusterRoles("verrazzano-monitor")
	addExec()

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Version: "1.2.0"},
			Status: vzapi.VerrazzanoStatus{
				State:      vzapi.VzStateUpgrading,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
			}},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
		&rancherIngress, &kcIngress, &authConfig, &kcSecret, &localAuthConfig, &firstLoginSetting,
		&verrazzanoAdminClusterRole, &verrazzanoMonitorClusterRole,
	).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconcileLoop(reconciler, request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)

	// check for upgrade completed condition
	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	found := false
	for _, condition := range verrazzano.Status.Conditions {
		if condition.Type == vzapi.CondUpgradeComplete {
			found = true
			break
		}
	}
	asserts.True(found, "expected upgrade completed to be true")
	assertKeycloakAuthConfig(asserts, spi.NewFakeContext(c, &verrazzano, nil, false))
}

// TestUpgradeCompletedMultipleReconcile tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile a verrazzano resource after install is completed
// WHEN spec.version doesn't match status.version and reconcile is called multiple times
// THEN ensure a condition with type UpgradeCompleted is added
func TestUpgradeCompletedMultipleReconcile(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	fname, _ := filepath.Abs(unitTestBomFile)
	config.SetDefaultBomFilePath(fname)
	asserts := assert.New(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	goClient, err := initFakeClient()
	asserts.NoError(err)
	k8sutil.SetFakeClient(goClient)

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{
				HelmComponent: helm2.HelmComponent{
					ReleaseName: "fake",
				},
			},
		}
	})
	defer registry.ResetGetComponentsFn()

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	_ = oamcore.AddToScheme(k8scheme.Scheme)

	authConfig := createKeycloakAuthConfig()
	localAuthConfig := createLocalAuthConfig()
	kcSecret := createKeycloakSecret()
	firstLoginSetting := createFirstLoginSetting()
	rancherIngress := createIngress(common.CattleSystem, constants.RancherIngress, common.RancherName)
	kcIngress := createIngress(constants.KeycloakNamespace, constants.KeycloakIngress, constants.KeycloakIngress)
	verrazzanoAdminClusterRole := createClusterRoles("verrazzano-admin")
	verrazzanoMonitorClusterRole := createClusterRoles("verrazzano-monitor")
	addExec()

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Version: "1.2.0"},
			Status: vzapi.VerrazzanoStatus{
				State:      vzapi.VzStateUpgrading,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
			}},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
		&rancherIngress, &kcIngress, &authConfig, &kcSecret, &localAuthConfig, &firstLoginSetting,
		&verrazzanoAdminClusterRole, &verrazzanoMonitorClusterRole,
	).Build()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconcileLoop(reconciler, request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)

	// check for upgrade completed condition
	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	found := false
	for _, condition := range verrazzano.Status.Conditions {
		if condition.Type == vzapi.CondUpgradeComplete {
			found = true
			break
		}
	}
	asserts.True(found, "expected upgrade completed to be true")
	assertKeycloakAuthConfig(asserts, spi.NewFakeContext(c, &verrazzano, nil, false))
}

// TestUpgradeCompletedStatusReturnsError tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN the update of the VZ resource status fails and returns an error
// THEN ensure an error is returned and a requeue is requested
func TestUpgradeCompletedStatusReturnsError(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	fname, _ := filepath.Abs(unitTestBomFile)
	config.SetDefaultBomFilePath(fname)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	goClient, err := initFakeClient()
	asserts.NoError(err)
	k8sutil.SetFakeClient(goClient)

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{},
		}
	})
	defer registry.ResetGetComponentsFn()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Expect a call to get the verrazzano resource.  Return resource with version
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:  name.Namespace,
				Name:       name.Name,
				Finalizers: []string{finalizerName}}
			verrazzano.Spec = vzapi.VerrazzanoSpec{
				Version: "1.2.0"}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				State:      vzapi.VzStateUpgrading,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
			}
			return nil
		}).AnyTimes()

	// Expect a call to get the service account
	expectGetServiceAccountExists(mock, name, nil)

	// Expect a call to get the ClusterRoleBinding
	expectClusterRoleBindingExists(mock, verrazzanoToUse, namespace, name)

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *oamapi.ApplicationConfigurationList, opts ...client.ListOption) error {
			return nil
		}).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return fmt.Errorf("Unexpected status error")
		}).AnyTimes()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconcileLoop(reconciler, request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
}

// TestUpgradeHelmError tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile a verrazzano resource after install is completed
// WHEN spec.version doesn't match status.version
// THEN ensure a condition with type UpgradePaused is added
func TestUpgradeHelmError(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{
				upgradeFunc: func(ctx spi.ComponentContext) error {
					return fmt.Errorf("Error running upgrade")
				},
			},
		}
	})

	defer registry.ResetGetComponentsFn()

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: func() metav1.ObjectMeta {
				om := createObjectMeta(namespace, name, []string{finalizerName})
				om.Generation = 1
				return om
			}(),
			Spec: vzapi.VerrazzanoSpec{
				Version: "0.2.0"},
			Status: vzapi.VerrazzanoStatus{
				State:      vzapi.VzStateUpgrading,
				Components: makeVerrazzanoComponentStatusMap(),
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconcileLoop(reconciler, request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.GreaterOrEqual(result.RequeueAfter.Seconds(), time.Duration(30).Seconds())

	// check for upgrade paused condition
	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	found := false
	for _, condition := range verrazzano.Status.Conditions {
		if condition.Type == vzapi.CondUpgradePaused {
			found = true
			break
		}
	}
	asserts.True(found, "expected upgrade paused to be true")
}

// TestUpgradeIsCompInstalledFailure tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an upgrade
// WHEN the comp.IsInstalled() function returns an error
// THEN an error is returned and the VZ status is not updated
func TestUpgradeIsCompInstalledFailure(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	vz := vzapi.Verrazzano{
		ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
		Spec: vzapi.VerrazzanoSpec{
			Version: "0.2.0"},
		Status: vzapi.VerrazzanoStatus{
			State:      vzapi.VzStateReady,
			Components: makeVerrazzanoComponentStatusMap(),
		},
	}

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vz,
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&vzapi.Verrazzano{}, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{
				isInstalledFunc: func(ctx spi.ComponentContext) (bool, error) {
					return false, fmt.Errorf("Error running isInstalled")
				},
			},
		}
	})
	defer registry.ResetGetComponentsFn()

	// Reconcile
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, _ := reconcileLoop(reconciler, request)

	// Validate the results
	asserts.Equal(true, result.Requeue)
}

// TestUpgradeComponent tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an upgrade
// WHEN the component upgrades normally
// THEN no error is returned and the correct spi.Component upgrade methods have been returned
func TestUpgradeComponent(t *testing.T) {
	initUnitTesing()
	// Need to use real component name since upgrade loops through registry
	componentName := oam.ComponentName

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	goClient, err := initFakeClient()
	asserts.NoError(err)
	k8sutil.SetFakeClient(goClient)

	vz := vzapi.Verrazzano{}
	vz.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	vz.ObjectMeta = createObjectMeta(namespace, name, []string{finalizerName})
	vz.Spec = vzapi.VerrazzanoSpec{
		Version: "1.2.0"}
	vz.Status = vzapi.VerrazzanoStatus{
		State: vzapi.VzStateUpgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondUpgradeStarted,
			},
		},
		Components: makeVerrazzanoComponentStatusMap(),
	}

	initStartingStates(&vz, componentName)

	mockComp := mocks.NewMockComponent(mocker)

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			mockComp,
		}
	})
	defer registry.ResetGetComponentsFn()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Set mock component expectations
	mockComp.EXPECT().IsInstalled(gomock.Any()).Return(true, nil).AnyTimes()
	mockComp.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()
	mockComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().Upgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().PostUpgrade(gomock.Any()).Return(nil).AnyTimes()
	mockComp.EXPECT().Name().Return(componentName).AnyTimes()
	mockComp.EXPECT().IsReady(gomock.Any()).Return(true).AnyTimes()

	ingressList := networkingv1.IngressList{Items: []networkingv1.Ingress{}}
	//sa := rbac.NewServiceAccount(namespace, name, []string{}, map[string]string{})
	//crb := rbac.NewClusterRoleBinding(&vz, buildClusterRoleBindingName(namespace, name), getInstallNamespace(), buildServiceAccountName(name))
	authConfig := createKeycloakAuthConfig()
	localAuthConfig := createLocalAuthConfig()
	kcSecret := createKeycloakSecret()
	firstLoginSetting := createFirstLoginSetting()
	rancherIngress := createIngress(common.CattleSystem, constants.RancherIngress, common.RancherName)
	kcIngress := createIngress(constants.KeycloakNamespace, constants.KeycloakIngress, constants.KeycloakIngress)
	verrazzanoAdminClusterRole := createClusterRoles("verrazzano-admin")
	verrazzanoMonitorClusterRole := createClusterRoles("verrazzano-monitor")
	addExec()

	appConfigList := oamapi.ApplicationConfigurationList{Items: []oamapi.ApplicationConfiguration{}}
	kcPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: "keycloak", Name: "dump-claim"},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz, &rancherIngress, &kcIngress, &authConfig, &kcSecret, &localAuthConfig, &firstLoginSetting, &verrazzanoAdminClusterRole, &verrazzanoMonitorClusterRole, kcPVC).WithLists(&ingressList, &appConfigList).Build()

	// Reconcile upgrade until state is done.  Put guard to prevent infinite loop
	reconciler := newVerrazzanoReconciler(c)
	numComponentStates := 10
	var result ctrl.Result
	for i := 0; i < numComponentStates; i++ {
		result, err = reconciler.reconcileUpgrade(vzlog.DefaultLogger(), &vz)
		if err != nil || !result.Requeue {
			break
		}
	}

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)

	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	asserts.NoError(err)

	asserts.Equal(vzapi.VzStateUpgrading, vz.Status.State)
	asserts.Equal(vz.Generation, vz.Status.Components[componentName].LastReconciledGeneration)
	asserts.Equal(vzapi.CondUpgradeStarted, vz.Status.Components[componentName].Conditions[1].Type)
	asserts.Equal(vzapi.CondUpgradeComplete, vz.Status.Components[componentName].Conditions[2].Type)
	assertKeycloakAuthConfig(asserts, spi.NewFakeContext(c, &vz, nil, false))

}

// TestUpgradeComponentWithBlockingStatus tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an upgrade
// WHEN the component fails to upgrade since a status other than "deployed" exists
// THEN the offending secret is deleted so the upgrade can proceed
func TestUpgradeComponentWithBlockingStatus(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	vz := vzapi.Verrazzano{}
	vz.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	vz.ObjectMeta = createObjectMeta(namespace, name, []string{finalizerName})
	vz.Spec = vzapi.VerrazzanoSpec{
		Version: "0.2.0"}
	vz.Status = vzapi.VerrazzanoStatus{
		State: vzapi.VzStateUpgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondUpgradeStarted,
			},
		},
		Components: makeVerrazzanoComponentStatusMap(),
	}

	mockComp := mocks.NewMockComponent(mocker)

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			mockComp,
		}
	})
	defer registry.ResetGetComponentsFn()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Set mock component expectations
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return nil
		}).AnyTimes()
	mockComp.EXPECT().IsInstalled(gomock.Any()).Return(true, nil).AnyTimes()
	mockComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(1)
	mockComp.EXPECT().Upgrade(gomock.Any()).Return(fmt.Errorf("Upgrade in progress")).AnyTimes()
	mockComp.EXPECT().Name().Return("testcomp").Times(1).AnyTimes()

	// expect a call to list any secrets with a status other than "deployed" for the component
	statuses := []string{"unknown", "uninstalled", "superseded", "failed", "uninstalling", "pending-install", "pending-upgrade", "pending-rollback"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(statuses))))
	status := statuses[int(n.Int64())]
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secretList *v1.SecretList, opts ...client.ListOption) error {
			secretList.Items = []v1.Secret{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": "testcomp", "status": status},
				},
			}}
			return nil
		}).AnyTimes()

	// expect a call to delete the secret
	mock.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return nil
		}).AnyTimes()

	// Reconcile upgrade
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconcileUpgradeLoop(reconciler, &vz)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
}

// TestUpgradeMultipleComponentsOneDisabled tests the reconcileUpgrade method for the following use case
// GIVEN a request to reconcile an upgrade
// WHEN where one component is enabled and another is disabled
// THEN the upgrade completes normally and the correct spi.Component upgrade methods have not been invoked for the disabled component
func TestUpgradeMultipleComponentsOneDisabled(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)
	mocker := gomock.NewController(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	goClient, err := initFakeClient()
	asserts.NoError(err)
	k8sutil.SetFakeClient(goClient)

	vz := vzapi.Verrazzano{}
	vz.TypeMeta = metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano"}
	vz.ObjectMeta = createObjectMeta(namespace, name, []string{finalizerName})
	vz.Spec = vzapi.VerrazzanoSpec{
		Version: "1.2.0"}
	vz.Status = vzapi.VerrazzanoStatus{
		State: vzapi.VzStateUpgrading,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondUpgradeStarted,
			},
		},
		Components: makeVerrazzanoComponentStatusMap(),
	}

	mockEnabledComp := mocks.NewMockComponent(mocker)
	mockDisabledComp := mocks.NewMockComponent(mocker)

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			mockEnabledComp,
			mockDisabledComp,
		}
	})
	defer registry.ResetGetComponentsFn()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Set enabled mock component expectations
	mockEnabledComp.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()
	mockEnabledComp.EXPECT().Name().Return("EnabledComponent").AnyTimes()
	mockEnabledComp.EXPECT().IsInstalled(gomock.Any()).Return(true, nil).AnyTimes()
	mockEnabledComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(1)
	mockEnabledComp.EXPECT().Upgrade(gomock.Any()).Return(nil).Times(1)
	mockEnabledComp.EXPECT().PostUpgrade(gomock.Any()).Return(nil).AnyTimes()
	mockEnabledComp.EXPECT().IsReady(gomock.Any()).Return(true).AnyTimes()
	mockEnabledComp.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()

	// Set disabled mock component expectations
	mockDisabledComp.EXPECT().Name().Return("DisabledComponent").Times(1).AnyTimes()
	mockDisabledComp.EXPECT().IsInstalled(gomock.Any()).Return(false, nil).AnyTimes()
	mockDisabledComp.EXPECT().PreUpgrade(gomock.Any()).Return(nil).Times(0)
	mockDisabledComp.EXPECT().Upgrade(gomock.Any()).Return(nil).Times(0)
	mockDisabledComp.EXPECT().PostUpgrade(gomock.Any()).Return(nil).AnyTimes()
	mockDisabledComp.EXPECT().IsEnabled(gomock.Any()).Return(false).AnyTimes()
	ingressList := networkingv1.IngressList{Items: []networkingv1.Ingress{}}

	authConfig := createKeycloakAuthConfig()
	localAuthConfig := createLocalAuthConfig()
	kcSecret := createKeycloakSecret()
	firstLoginSetting := createFirstLoginSetting()
	rancherIngress := createIngress(common.CattleSystem, constants.RancherIngress, common.RancherName)
	kcIngress := createIngress(constants.KeycloakNamespace, constants.KeycloakIngress, constants.KeycloakIngress)
	verrazzanoAdminClusterRole := createClusterRoles("verrazzano-admin")
	verrazzanoMonitorClusterRole := createClusterRoles("verrazzano-monitor")
	addExec()

	appConfigList := oamapi.ApplicationConfigurationList{Items: []oamapi.ApplicationConfiguration{}}
	kcPVC := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: "keycloak", Name: "dump-claim"},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(&vz, &rancherIngress, &kcIngress, &authConfig, &kcSecret, &localAuthConfig, &firstLoginSetting, &verrazzanoAdminClusterRole, &verrazzanoMonitorClusterRole, kcPVC).WithLists(&ingressList, &appConfigList).Build()

	// Reconcile upgrade until state is done.  Put guard to prevent infinite loop
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconcileUpgradeLoop(reconciler, &vz)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)

	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	asserts.NoError(err)

	asserts.Equal(vzapi.VzStateUpgrading, vz.Status.State)
	asserts.Equal(vz.Generation, vz.Status.Components[mockEnabledComp.Name()].LastReconciledGeneration)
	asserts.Equal(vzapi.CondUpgradeStarted, vz.Status.Components[mockEnabledComp.Name()].Conditions[0].Type)
	asserts.Equal(vzapi.CondUpgradeComplete, vz.Status.Components[mockEnabledComp.Name()].Conditions[1].Type)
	asserts.Equal(int64(0), vz.Status.Components[mockEnabledComp.Name()].LastReconciledGeneration)
	asserts.Equal(false, result.Requeue)
	assertKeycloakAuthConfig(asserts, spi.NewFakeContext(c, &vz, nil, false))
}

// TestRetryUpgrade tests the retryUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after a failed upgrade
// WHEN the restart-version annotation and the observed-restart-version annotation don't match and
// WHEN spec.version doesn't match status.version
// THEN ensure the annotations are updated and the reconciler requeues with the Ready StateType
func TestRetryUpgrade(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: func() metav1.ObjectMeta {
				om := createObjectMeta(namespace, name, []string{finalizerName})
				om.Annotations = map[string]string{constants.UpgradeRetryVersion: "a"}
				return om
			}(),
			Spec: vzapi.VerrazzanoSpec{
				Version: "0.2.0"},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStateFailed,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondUpgradeFailed,
					},
				},
				Components: makeVerrazzanoComponentStatusMap(),
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(1), result.RequeueAfter)

	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	asserts.Equal(verrazzano.ObjectMeta.Annotations[constants.UpgradeRetryVersion], "a", "Incorrect restart version")
	asserts.Equal(verrazzano.ObjectMeta.Annotations[constants.ObservedUpgradeRetryVersion], "a", "Incorrect observed restart version")
	asserts.Len(verrazzano.Status.Conditions, 1, "Incorrect number of conditions")
	asserts.Equal(verrazzano.Status.State, vzapi.VzStateReady, "Incorrect State")
}

// TestTransitionToPausedUpgradeFromFailed tests the pause of an upgrade for the following use case
// GIVEN a request to reconcile an verrazzano resource after a failed upgrade
// WHEN the VPO version is not the verrazzano version
// THEN ensure the reconciler transitions to the paused StateType
func TestTransitionToPausedUpgradeFromFailed(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Version: "1.0.0"},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStateFailed,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondUpgradeFailed,
					},
				},
				Components: makeVerrazzanoComponentStatusMap(),
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(2)*time.Second, result.RequeueAfter)

	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	asserts.Len(verrazzano.Status.Conditions, 2, "Incorrect number of conditions")
	asserts.Equal(verrazzano.Status.State, vzapi.VzStatePaused, "Incorrect State")

}

// TestTransitionToPausedUpgradeFromStarted tests the pause of an upgrade for the following use case
// GIVEN a request to reconcile an verrazzano resource during an upgrade that is in progress
// WHEN the VPO version is not the verrazzano version
// THEN ensure the reconciler transitions to a paused StateType
func TestTransitionToPausedUpgradeFromStarted(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Version: "1.0.0"},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStateUpgrading,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondUpgradeStarted,
					},
				},
				Components: makeVerrazzanoComponentStatusMap(),
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.Equal(time.Duration(2)*time.Second, result.RequeueAfter)

	verrazzano := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &verrazzano)
	asserts.NoError(err)
	asserts.Len(verrazzano.Status.Conditions, 2, "Incorrect number of conditions")
	asserts.Equal(verrazzano.Status.State, vzapi.VzStatePaused, "Incorrect State")
}

// TestTransitionFromPausedUpgrade tests the resumption of an upgrade for the following use case
// GIVEN a request to reconcile an verrazzano resource when it is paused
// WHEN the VPO version matches the verrazzano version
// THEN ensure the reconciler transitions to a ready StateType
func TestTransitionFromPausedUpgrade(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Version: "1.0.1"},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStatePaused,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondUpgradePaused,
					},
				},
				Components: makeVerrazzanoComponentStatusMap(),
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.GreaterOrEqual(result.RequeueAfter.Seconds(), time.Duration(30).Seconds())

	// Get the resulting VZ resource
	vz := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	asserts.NoError(err)
	asserts.Len(vz.Status.Conditions, 1, "Incorrect number of conditions")
	asserts.Equal(vz.Status.State, vzapi.VzStateReady, "Incorrect State")
}

// TestDontRetryUpgrade tests the retryUpgrade method for the following use case
// GIVEN a request to reconcile an verrazzano resource after a failed upgrade
// WHEN the restart-version annotation and the observed-restart-version annotation match and
// THEN ensure that
func TestDontRetryUpgrade(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: func() metav1.ObjectMeta {
				om := createObjectMeta(namespace, name, []string{finalizerName})
				om.Annotations = map[string]string{constants.UpgradeRetryVersion: "b", constants.ObservedUpgradeRetryVersion: "b"}
				return om
			}(),
			Spec: vzapi.VerrazzanoSpec{
				Version: "1.1.0"},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStateFailed,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondUpgradeFailed,
					},
				},
				Components: makeVerrazzanoComponentStatusMap(),
			},
		},
		rbac.NewServiceAccount(namespace, name, []string{}, nil),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.True(result.IsZero())
}

// TestIsLastConditionNone tests the isLastCondition method for the following use case
// GIVEN an empty array of conditions
// WHEN isLastCondition is called
// THEN ensure that false
func TestIsLastConditionNone(t *testing.T) {
	asserts := assert.New(t)
	asserts.False(isLastCondition(vzapi.VerrazzanoStatus{}, vzapi.CondUpgradeComplete), "isLastCondition should have returned false")
}

// TestIsLastConditionFalse tests the isLastCondition method for the following use case
// GIVEN an array of conditions
// WHEN isLastCondition is called where the target last condition doesn't match the actual last condition
// THEN ensure that false is returned
func TestIsLastConditionFalse(t *testing.T) {
	asserts := assert.New(t)
	st := vzapi.VerrazzanoStatus{
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondUpgradeComplete,
			},
			{
				Type: vzapi.CondInstallFailed,
			},
		},
	}
	asserts.False(isLastCondition(st, vzapi.CondUpgradeComplete), "isLastCondition should have returned false")
}

// TestIsLastConditionTrue tests the isLastCondition method for the following use case
// GIVEN an array of conditions
// WHEN isLastCondition is called where the target last condition matches the actual last condition
// THEN ensure that true is returned
func TestIsLastConditionTrue(t *testing.T) {
	asserts := assert.New(t)
	st := vzapi.VerrazzanoStatus{
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondUpgradeComplete,
			},
			{
				Type: vzapi.CondInstallFailed,
			},
		},
	}
	asserts.True(isLastCondition(st, vzapi.CondInstallFailed), "isLastCondition should have returned true")
}

// TestInstanceRestoreWithEmptyStatus tests the reconcileUpdate method for the following use case
// WHEN instance is restored via backup and restore instance status is not updated
// WHEN components are already installed.
// When verrazzano instance status is empty
// THEN update the instance urls appropriately
func TestInstanceRestoreWithEmptyStatus(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{}

	keycloakEnabled := false
	verrazzanoToUse := vzapi.Verrazzano{
		ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Keycloak: &vzapi.KeycloakComponent{
					Enabled: &keycloakEnabled,
				},
				Istio: &vzapi.IstioComponent{
					Enabled: &istioEnabled,
				},
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &jaegerEnabled,
				},
			},
		},
		Status: vzapi.VerrazzanoStatus{
			State: vzapi.VzStateReady,
			Conditions: []vzapi.Condition{
				{
					Type: vzapi.CondInstallComplete,
				},
			},
			Components: makeVerrazzanoComponentStatusMap(),
		},
	}
	verrazzanoToUse.Status.Components[keycloak.ComponentName].State = vzapi.CompStateDisabled
	verrazzanoToUse.Status.Components[istio.ComponentName].State = vzapi.CompStateDisabled

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&verrazzanoToUse,
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: "cattle-system", Name: "rancher"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: rancherURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: "keycloak", Name: "keycloak"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: keycloakURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-es-ingest"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: esURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-prometheus"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: promURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-grafana"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: grafanaURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: kialiURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kibana"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: kibanaURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: consoleURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.JaegerIngress},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: jaegerURL},
				},
			},
		},
		rbac.NewServiceAccount(getInstallNamespace(), buildServiceAccountName(name), []string{}, labels),
		rbac.NewClusterRoleBinding(
			&verrazzanoToUse,
			buildClusterRoleBindingName(namespace, name),
			getInstallNamespace(),
			buildServiceAccountName(buildClusterRoleBindingName(namespace, name)))).Build()

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stub-out the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stub-out the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)

	// validating instance urls are updated
	// Status is empty in this case
	enabled := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Console: &vzapi.ConsoleComponent{
					Enabled: &enabled,
				},
			},
		},
	}
	instanceInfo := vzinstance.GetInstanceInfo(spi.NewFakeContext(c, vz, nil, false))
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, "https://"+consoleURL, *instanceInfo.ConsoleURL)
	assert.Equal(t, "https://"+rancherURL, *instanceInfo.RancherURL)
	assert.Equal(t, "https://"+keycloakURL, *instanceInfo.KeyCloakURL)
	assert.Equal(t, "https://"+esURL, *instanceInfo.ElasticURL)
	assert.Equal(t, "https://"+grafanaURL, *instanceInfo.GrafanaURL)
	assert.Equal(t, "https://"+kialiURL, *instanceInfo.KialiURL)
	assert.Equal(t, "https://"+kibanaURL, *instanceInfo.KibanaURL)
	assert.Equal(t, "https://"+promURL, *instanceInfo.PrometheusURL)
	assert.Equal(t, "https://"+jaegerURL, *instanceInfo.JaegerURL)
}

// TestInstanceRestoreWithPopulatedStatus tests the reconcileUpdate method for the following use case
// WHEN instance is restored via backup and restore instance status is not updated
// WHEN components are already installed.
// When verrazzano instance status is not empty
// THEN update the instance urls appropriately
func TestInstanceRestoreWithPopulatedStatus(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	keycloakEnabled := false
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Keycloak: &vzapi.KeycloakComponent{
						Enabled: &keycloakEnabled,
					},
					Istio: &vzapi.IstioComponent{
						Enabled: &istioEnabled,
					},
				},
			},
			Status: vzapi.VerrazzanoStatus{
				State: vzapi.VzStateReady,
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.CondInstallComplete,
					},
				},
				Components: func() vzapi.ComponentStatusMap {
					statusMap := makeVerrazzanoComponentStatusMap()
					statusMap[keycloak.ComponentName].State = vzapi.CompStateDisabled
					statusMap[istio.ComponentName].State = vzapi.CompStateDisabled
					return statusMap
				}(),
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: "cattle-system", Name: "rancher"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: rancherURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: "keycloak", Name: "keycloak"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: keycloakURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-es-ingest"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: esURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-prometheus"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: promURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-grafana"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: grafanaURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: kialiURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kibana"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: kibanaURL},
				},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: constants.VzConsoleIngress},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{Host: consoleURL},
				},
			},
		},
		rbac.NewServiceAccount(name, namespace, []string{}, labels),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
	).Build()

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stub-out the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	// Stub-out the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)

	// validating instance urls are updated
	fakeInstanceInfo := vzapi.InstanceInfo{}
	consolefqdn := fmt.Sprintf("https://%s", consoleURL)
	fakeInstanceInfo.ConsoleURL = &consolefqdn

	rancherfqdn := fmt.Sprintf("https://%s", rancherURL)
	fakeInstanceInfo.ConsoleURL = &rancherfqdn

	keycloakfqdn := fmt.Sprintf("https://%s", keycloakURL)
	fakeInstanceInfo.ConsoleURL = &keycloakfqdn

	esfqdn := fmt.Sprintf("https://%s", esURL)
	fakeInstanceInfo.ConsoleURL = &esfqdn

	grafanafqdn := fmt.Sprintf("https://%s", grafanaURL)
	fakeInstanceInfo.ConsoleURL = &grafanafqdn

	kibanafqdn := fmt.Sprintf("https://%s", kibanaURL)
	fakeInstanceInfo.ConsoleURL = &kibanafqdn

	kialifqdn := fmt.Sprintf("https://%s", kialiURL)
	fakeInstanceInfo.ConsoleURL = &kialifqdn

	promfqdn := fmt.Sprintf("https://%s", promURL)
	fakeInstanceInfo.ConsoleURL = &promfqdn

	enabled := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Console: &vzapi.ConsoleComponent{
					Enabled: &enabled,
				},
			},
		},
	}
	vz.Status.VerrazzanoInstance = &fakeInstanceInfo

	instanceInfo := vzinstance.GetInstanceInfo(spi.NewFakeContext(c, vz, nil, false))
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, "https://"+consoleURL, *instanceInfo.ConsoleURL)
	assert.Equal(t, "https://"+rancherURL, *instanceInfo.RancherURL)
	assert.Equal(t, "https://"+keycloakURL, *instanceInfo.KeyCloakURL)
	assert.Equal(t, "https://"+esURL, *instanceInfo.ElasticURL)
	assert.Equal(t, "https://"+grafanaURL, *instanceInfo.GrafanaURL)
	assert.Equal(t, "https://"+kialiURL, *instanceInfo.KialiURL)
	assert.Equal(t, "https://"+kibanaURL, *instanceInfo.KibanaURL)
	assert.Equal(t, "https://"+promURL, *instanceInfo.PrometheusURL)
}

// initStartingStates inits the starting state for verrazzano and component upgrade
func initStartingStates(cr *vzapi.Verrazzano, compName string) {
	initStates(cr, vzStateStart, compName, compStateUpgradeInit)
}

// initStates inits the specified state for verrazzano and component upgrade
func initStates(cr *vzapi.Verrazzano, vzState VerrazzanoUpgradeState, compName string, compState componentUpgradeState) {
	tracker := getUpgradeTracker(cr)
	tracker.vzState = vzState
	upgradeContext := tracker.getComponentUpgradeContext(compName)
	upgradeContext.state = compState
}

// reconcileUpgradeLoop
func reconcileUpgradeLoop(reconciler Reconciler, cr *vzapi.Verrazzano) (ctrl.Result, error) {
	numComponentStates := 7
	var err error
	var result ctrl.Result
	for i := 0; i < numComponentStates; i++ {
		result, err = reconciler.reconcileUpgrade(vzlog.DefaultLogger(), cr)
		if err != nil || !result.Requeue {
			break
		}
	}
	return result, err
}

// reconcileLoop
func reconcileLoop(reconciler Reconciler, request ctrl.Request) (ctrl.Result, error) {
	numComponentStates := 20
	var err error
	var result ctrl.Result
	for i := 0; i < numComponentStates; i++ {
		result, err = reconciler.Reconcile(context.TODO(), request)
		if err != nil || !result.Requeue {
			break
		}
	}
	return result, err
}

// initFakeClient inits a fake go-client and loads it with fake resources
func initFakeClient() (kubernetes.Interface, error) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testDeployment",
			Namespace: "verrazzano-system",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testPod",
			Namespace: "verrazzano-system",
			Labels:    map[string]string{"app": "foo"},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  "c1",
				Image: "myimage",
			}},
		},
	}
	clientSet := gofake.NewSimpleClientset(dep, pod)
	return clientSet, nil
}

func createObjectMeta(namespace string, name string, finalizers []string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace:  namespace,
		Name:       name,
		Finalizers: finalizers,
	}
}
