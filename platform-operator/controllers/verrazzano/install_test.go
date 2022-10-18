// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/rbac"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testBomFile = "../../verrazzano-bom.json"
const fakeCompReleaseName = "verrazzano-authproxy"

var (
	namespace                = "verrazzano"
	name                     = "test-verrazzano"
	falseVal                 = false
	statusVer                = "1.3.0"
	lastReconciledGeneration = int64(2)
	reconcilingGen           = int64(0)
)

// TestStartUpdate tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a Reconciling State
func TestStartUpdate(t *testing.T) {
	initUnitTesing()
	status := vzapi.VerrazzanoStatus{
		State:   vzapi.VzStateReady,
		Version: statusVer,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondInstallStarted,
			},
			{
				Type: vzapi.CondInstallComplete,
			},
		},
	}

	ctx, asserts, result, fakeCompUpdated, err := testUpdate(t,
		lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", status, namespace, name, "true")
	asserts.NoError(err)

	defer reset()

	vz := vzapi.Verrazzano{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	asserts.NoError(err)

	asserts.Equal(vzapi.VzStateReconciling, vz.Status.State)
	asserts.False(*fakeCompUpdated)
	asserts.True(result.Requeue)
}

// TestUpdate tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install has been started
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a Ready State
func TestCompleteUpdate(t *testing.T) {
	initUnitTesing()
	status := vzapi.VerrazzanoStatus{
		State:   vzapi.VzStateReconciling,
		Version: statusVer,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondInstallComplete,
			},
			{
				Type: vzapi.CondInstallStarted,
			},
		},
	}
	ctx, asserts, result, fakeCompUpdated, err := testUpdate(t,
		lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", status, namespace, name, "true")
	asserts.NoError(err)

	defer reset()

	vz := vzapi.Verrazzano{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	asserts.NoError(err)

	asserts.Equal(vzapi.VzStateReady, vz.Status.State)
	asserts.True(*fakeCompUpdated)
	asserts.Equal(vz.Generation, vz.Status.Components[fakeCompReleaseName].LastReconciledGeneration)
	asserts.Equal(vzapi.CondInstallStarted, vz.Status.Components[fakeCompReleaseName].Conditions[0].Type)
	asserts.Equal(vzapi.CondInstallComplete, vz.Status.Components[fakeCompReleaseName].Conditions[1].Type)
	asserts.False(result.Requeue)
	assertKeycloakAuthConfig(asserts, ctx)
}

// TestNoUpdateSameGeneration tests the reconcile func with same generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the same LastReconciledGeneration as verrazzano CR
// THEN ensure a Ready State
func TestNoUpdateSameGeneration(t *testing.T) {
	initUnitTesing()
	status := vzapi.VerrazzanoStatus{
		State:   vzapi.VzStateReady,
		Version: statusVer,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondInstallComplete,
			},
		},
	}

	ctx, asserts, result, fakeCompUpdated, err := testUpdate(t,
		lastReconciledGeneration, reconcilingGen, lastReconciledGeneration,
		"1.3.0", status, namespace, name, "true")
	asserts.NoError(err)

	defer reset()

	vz := vzapi.Verrazzano{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	asserts.NoError(err)

	asserts.Equal(vzapi.VzStateReady, vz.Status.State)
	asserts.False(*fakeCompUpdated)
	asserts.False(result.Requeue)
}

// TestUpdateWithUpgrade tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure an Upgrading State
func TestUpdateWithUpgrade(t *testing.T) {
	initUnitTesing()
	status := vzapi.VerrazzanoStatus{
		State:   vzapi.VzStateReady,
		Version: "1.2.0",
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondInstallComplete,
			},
		},
	}

	ctx, asserts, result, fakeCompUpdated, err := testUpdate(t,
		lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", status, namespace, name, "true")
	asserts.NoError(err)

	defer reset()

	vz := vzapi.Verrazzano{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	asserts.NoError(err)

	asserts.NoError(err)
	asserts.Equal(vzapi.VzStateUpgrading, vz.Status.State)
	asserts.False(*fakeCompUpdated)
	asserts.True(result.Requeue)
}

// TestUpdateOnUpdate tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration than verrazzano CR in the request
// THEN ensure a Reconciling State
func TestUpdateOnUpdate(t *testing.T) {
	initUnitTesing()
	reconcilingGeneration := int64(3)

	status := vzapi.VerrazzanoStatus{
		State:   vzapi.VzStateReady,
		Version: statusVer,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondInstallStarted,
			},
			{
				Type: vzapi.CondInstallComplete,
			},
		},
	}

	ctx, asserts, result, fakeCompUpdated, err := testUpdate(t,
		reconcilingGeneration+1, reconcilingGeneration, lastReconciledGeneration,
		"1.3.0", status, namespace, name, "true")
	asserts.NoError(err)

	defer reset()

	vz := vzapi.Verrazzano{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	asserts.NoError(err)

	asserts.Equal(vzapi.VzStateReconciling, vz.Status.State)
	asserts.False(*fakeCompUpdated)
	asserts.True(result.Requeue)
}

// TestUpdateFalseMonitorChanges tests the reconcile func with updated generation
// GIVEN a request to reconcile an verrazzano resource after install is completed
// WHEN all components have the smaller LastReconciledGeneration but MonitorOverrides returns false
// THEN ensure a Ready State
func TestUpdateFalseMonitorChanges(t *testing.T) {
	initUnitTesing()
	status := vzapi.VerrazzanoStatus{
		State:   vzapi.VzStateReconciling,
		Version: statusVer,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondInstallComplete,
			},
			{
				Type: vzapi.CondInstallStarted,
			},
		},
	}
	ctx, asserts, result, fakeCompUpdated, err := testUpdate(t,
		lastReconciledGeneration+1, reconcilingGen, lastReconciledGeneration,
		"1.3.0", status, namespace, name, "false")
	asserts.NoError(err)

	defer reset()

	vz := vzapi.Verrazzano{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &vz)
	asserts.NoError(err)

	asserts.Equal(vzapi.VzStateReady, vz.Status.State)
	asserts.False(*fakeCompUpdated)
	asserts.Equal(lastReconciledGeneration, vz.Status.Components[fakeCompReleaseName].LastReconciledGeneration)
	asserts.False(result.Requeue)
	assertKeycloakAuthConfig(asserts, ctx)
}

func reset() {
	registry.ResetGetComponentsFn()
	config.SetDefaultBomFilePath("")
	helm.SetDefaultChartStatusFunction()
	config.SetDefaultBomFilePath("")
	helm.SetDefaultChartStatusFunction()
	config.TestProfilesDir = ""
}

// testUpdate creates a fake client and calls Reconcile to test update behavior
func testUpdate(t *testing.T,
	vzCrGen, reconcilingGen, lastReconciledGeneration int64,
	specVer string, vzStatus vzapi.VerrazzanoStatus, namespace, name, monitorChanges string) (spi.ComponentContext, *assert.Assertions, ctrl.Result, *bool, error) {

	asserts := assert.New(t)

	config.SetDefaultBomFilePath(testBomFile)

	fakeComp := fakeComponent{}
	fakeComp.ReleaseName = fakeCompReleaseName
	fakeComp.SupportsOperatorInstall = true
	fakeComp.monitorChanges = monitorChanges
	fakeCompUpdated := &falseVal
	fakeComp.installFunc = func(ctx spi.ComponentContext) error {
		update := true
		fakeCompUpdated = &update
		return nil
	}
	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComp,
		}
	})
	compStatusMap := makeVerrazzanoComponentStatusMap()
	available := true
	for _, status := range compStatusMap {
		status.ReconcilingGeneration = reconcilingGen
		status.LastReconciledGeneration = lastReconciledGeneration
		status.Available = &available
	}

	vzStatus.Components = compStatusMap

	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "install.verrazzano.io/v1alpha1",
			Kind:       "Verrazzano",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  namespace,
			Name:       name,
			Generation: vzCrGen,
			Finalizers: []string{finalizerName},
		},
		Spec: vzapi.VerrazzanoSpec{
			Version: specVer,
		},
		Status: vzStatus,
	}

	ingressList := networkingv1.IngressList{Items: []networkingv1.Ingress{}}
	sa := rbac.NewServiceAccount(namespace, name, []string{}, map[string]string{})
	crb := rbac.NewClusterRoleBinding(vz, buildClusterRoleBindingName(namespace, name), getInstallNamespace(), buildServiceAccountName(name))
	authConfig := createKeycloakAuthConfig()
	localAuthConfig := createLocalAuthConfig()
	kcSecret := createKeycloakSecret()
	firstLoginSetting := createFirstLoginSetting()
	rancherIngress := createIngress(common.CattleSystem, constants.RancherIngress, common.RancherName)
	kcIngress := createIngress(constants.KeycloakNamespace, constants.KeycloakIngress, constants.KeycloakIngress)
	verrazzanoAdminClusterRole := createClusterRoles("verrazzano-admin")
	verrazzanoMonitorClusterRole := createClusterRoles("verrazzano-monitor")
	addExec()

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vz, sa, crb, &rancherIngress, &kcIngress, &authConfig, &kcSecret, &localAuthConfig, &firstLoginSetting, &verrazzanoAdminClusterRole, &verrazzanoMonitorClusterRole).WithLists(&ingressList).Build()

	ctx := spi.NewFakeContext(c, vz, nil, false)
	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})

	// Sample bom file for version validation functions
	config.SetDefaultBomFilePath(testBomFilePath)
	// Stubout the call to check the chart status
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	config.TestProfilesDir = "../../manifests/profiles"

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(ctx.Client())
	result, err := reconciler.Reconcile(context.TODO(), request)
	return ctx, asserts, result, fakeCompUpdated, err
}

func createKeycloakAuthConfig() unstructured.Unstructured {
	authConfig := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	authConfig.SetGroupVersionKind(common.GVKAuthConfig)
	authConfig.SetName(common.AuthConfigKeycloak)
	return authConfig
}

func createLocalAuthConfig() unstructured.Unstructured {
	authConfig := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	authConfig.SetGroupVersionKind(common.GVKAuthConfig)
	authConfig.SetName(rancher.AuthConfigLocal)
	return authConfig
}

func createKeycloakSecret() corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "keycloak",
			Name:      "keycloak-http",
		},
		Data: map[string][]byte{
			"password": []byte("blahblah"),
		},
	}
}

func createFirstLoginSetting() unstructured.Unstructured {
	firstLoginSetting := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	firstLoginSetting.SetGroupVersionKind(common.GVKSetting)
	firstLoginSetting.SetName(common.SettingFirstLogin)
	return firstLoginSetting
}

func createClusterRoles(roleName string) rbacv1.ClusterRole {
	return rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: roleName}}
}

func createIngress(namespace, name, host string) networkingv1.Ingress {
	return networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name,
			Annotations: map[string]string{},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
				},
			},
		},
	}
}

func addExec() {
	scheme.Scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"}, &corev1.PodExecOptions{})
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodExecResult = func(url *url.URL) (string, string, error) {
		var commands []string
		if commands = url.Query()["command"]; len(commands) == 3 {
			if strings.Contains(commands[2], "id,clientId") {
				return "[{\"id\":\"something\", \"clientId\":\"" + rancher.AuthConfigKeycloakClientIDRancher + "\"}]", "", nil
			}

			if strings.Contains(commands[2], "client-secret") {
				return "{\"type\":\"secret\",\"value\":\"abcdef\"}", "", nil
			}

			if strings.Contains(commands[2], "get users") {
				return "[{\"id\":\"something\", \"username\":\"verrazzano\"}]", "", nil
			}

		}
		return "", "", nil
	}
}

func assertKeycloakAuthConfig(asserts *assert.Assertions, ctx spi.ComponentContext) {
	authConfig := createKeycloakAuthConfig()
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: common.AuthConfigKeycloak}, &authConfig)
	authConfigData := authConfig.UnstructuredContent()
	asserts.Nil(err)
	asserts.Equal(authConfigData[rancher.AuthConfigKeycloakAttributeRancherURL], fmt.Sprintf("https://%s%s", constants.RancherIngress, rancher.AuthConfigKeycloakURLPathVerifyAuth))
	asserts.Equal(authConfigData[rancher.AuthConfigKeycloakAttributeAuthEndpoint], fmt.Sprintf("https://%s%s", constants.KeycloakIngress, rancher.AuthConfigKeycloakURLPathAuthEndPoint))
	asserts.Equal(authConfigData[rancher.AuthConfigKeycloakAttributeClientID], rancher.AuthConfigKeycloakClientIDRancher)
}
