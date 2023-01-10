// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"reflect"
	"testing"
	"time"

	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/console"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	helm2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	jaegeroperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/jaeger/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearchdashboards"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/velero"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/rbac"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestReconcileUninstalling tests the reconcileUninstall method for the following use case
// GIVEN a request to reconcile a verrazzano resource to uninstall
// WHEN reconcileUninstall is called
// THEN ensure the component goes into the uninstalling state
func TestReconcileUninstalling(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	config.TestProfilesDir = relativeProfilesDir
	defer func() { config.TestProfilesDir = "" }()

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{
				HelmComponent: helm2.HelmComponent{
					ReleaseName:               "fake",
					SupportsOperatorUninstall: true,
				},
			},
		}
	})
	defer registry.ResetGetComponentsFn()

	k8sutil.GetCoreV1Func = common.MockGetCoreV1()
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()
	defer func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
	}()

	vzcr := &vzapi.Verrazzano{
		ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{},
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
				return statusMap
			}(),
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	_ = v1alpha1.AddToScheme(k8scheme.Scheme)
	_ = vzappclusters.AddToScheme(k8scheme.Scheme)

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		vzcr,
		rbac.NewServiceAccount(namespace, name, []string{}, labels),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: keycloak.ComponentNamespace,
				Name:      keycloak.ComponentName,
				Labels:    map[string]string{"app": keycloak.ComponentName},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	).Build()

	reconciler := newVerrazzanoReconciler(c)
	DeleteUninstallTracker(vzcr)
	result, err := reconciler.reconcileUninstall(vzlog.DefaultLogger(), vzcr)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, vzcr)
	asserts.NoError(err)
	asserts.NotZero(len(vzcr.Status.Components), "Status.Components len should not be zero")
	asserts.Equal("Uninstalling", string(vzcr.Status.Components["fake"].State), "Invalid component state")
	asserts.NotZero(len(UninstallTrackerMap), "UninstallTrackerMap should have no entries")
}

// TestReconcileUninstall tests the reconcileUninstall method for the following use case
// GIVEN a request to reconcile a verrazzano resource to uninstall
// WHEN reconcileUninstall is called
// THEN ensure the reconcileUninstall executes all states
func TestReconcileUninstall(t *testing.T) {
	initUnitTesing()
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	config.SetDefaultBomFilePath(unitTestBomFile)
	asserts := assert.New(t)

	config.TestProfilesDir = relativeProfilesDir
	defer func() { config.TestProfilesDir = "" }()

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{
				HelmComponent: helm2.HelmComponent{
					ReleaseName:               "fake",
					SupportsOperatorUninstall: true,
				},
			},
		}
	})

	defer registry.ResetGetComponentsFn()

	k8sutil.GetCoreV1Func = common.MockGetCoreV1()
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()
	defer func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
	}()

	vzcr := &vzapi.Verrazzano{
		ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{},
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
				return statusMap
			}(),
		},
	}
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	_ = v1alpha1.AddToScheme(k8scheme.Scheme)
	_ = vzappclusters.AddToScheme(k8scheme.Scheme)

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		vzcr,
		rbac.NewServiceAccount(namespace, name, []string{}, labels),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: keycloak.ComponentNamespace,
				Name:      keycloak.ComponentName,
				Labels:    map[string]string{"app": keycloak.ComponentName},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	).Build()

	// call reconcile once with installed true, then again with installed false
	reconciler := newVerrazzanoReconciler(c)
	DeleteUninstallTracker(vzcr)
	result, err := reconciler.reconcileUninstall(vzlog.DefaultLogger(), vzcr)
	asserts.NoError(err)
	asserts.Equal(true, result.Requeue)
	asserts.NotEqual(time.Duration(0), result.RequeueAfter)

	registry.OverrideGetComponentsFn(func() []spi.Component {
		return []spi.Component{
			fakeComponent{
				HelmComponent: helm2.HelmComponent{
					ReleaseName:               "fake",
					SupportsOperatorUninstall: true,
				},
				isInstalledFunc: func(ctx spi.ComponentContext) (bool, error) {
					return false, nil
				},
			},
		}
	})
	// reconcile a second time
	result, err = reconciler.reconcileUninstall(vzlog.DefaultLogger(), vzcr)
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, vzcr)
	asserts.NoError(err)
	asserts.NotZero(len(vzcr.Status.Components), "Status.Components len should not be zero")
	asserts.Equal("Uninstalled", string(vzcr.Status.Components["fake"].State), "Invalid component state")
	asserts.NotZero(len(UninstallTrackerMap), "UninstallTrackerMap should have no entries")
}

// TestUninstallVariations tests the reconcileUninstall method for the following use case
// GIVEN a request to reconcile a verrazzano resource to uninstall
// WHEN reconcileUninstall is called
// THEN ensure the reconcileUninstall cleans up all resources, including MC resources
func TestUninstallVariations(t *testing.T) {
	tests := []struct {
		name              string
		managed           bool
		createMCNamespace bool
		createProject     bool
		secrets           []corev1.Secret
	}{
		{
			name:              "no-mc",
			createMCNamespace: false,
		},
		// Admin cluster test, MC namespace should get deleted
		{
			name:              "admin-cluster",
			managed:           false,
			createMCNamespace: true,
			secrets: []corev1.Secret{
				{ObjectMeta: metav1.ObjectMeta{Name: vzconst.MCRegistrationSecret, Namespace: vzconst.VerrazzanoSystemNamespace}},
				{ObjectMeta: metav1.ObjectMeta{Name: mcElasticSearchScrt, Namespace: vzconst.VerrazzanoSystemNamespace}},
			},
		},
		// Admin cluster test with project, MC namespace should NOT get deleted
		{
			name:              "admin-cluster-with-projects",
			managed:           false,
			createMCNamespace: true,
			createProject:     true,
		},
		// Managed cluster test
		{
			name:              "managed-cluster",
			managed:           true,
			createMCNamespace: true,
			secrets: []corev1.Secret{
				{ObjectMeta: metav1.ObjectMeta{Name: vzconst.MCAgentSecret, Namespace: vzconst.VerrazzanoSystemNamespace}},
				{ObjectMeta: metav1.ObjectMeta{Name: vzconst.MCRegistrationSecret, Namespace: vzconst.VerrazzanoSystemNamespace}},
				{ObjectMeta: metav1.ObjectMeta{Name: mcElasticSearchScrt, Namespace: vzconst.VerrazzanoSystemNamespace}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			namespace := "verrazzano"
			name := "test"

			initUnitTesing()

			config.SetDefaultBomFilePath(unitTestBomFile)
			asserts := assert.New(t)

			config.TestProfilesDir = relativeProfilesDir
			defer func() { config.TestProfilesDir = "" }()

			defer config.Set(config.Get())
			config.Set(config.OperatorConfig{VersionCheckEnabled: false})

			_ = vzapi.AddToScheme(k8scheme.Scheme)
			_ = v1alpha1.AddToScheme(k8scheme.Scheme)
			_ = vzappclusters.AddToScheme(k8scheme.Scheme)

			c, vzcr := buildFakeClientAndObjects(test.createMCNamespace, test.createProject, test.secrets)

			// Delete tracker so each test has a fresh state machine
			DeleteUninstallTracker(vzcr)

			// reconcile a first time with isInstalled returning true
			registry.OverrideGetComponentsFn(func() []spi.Component {
				return []spi.Component{
					fakeComponent{
						HelmComponent: helm2.HelmComponent{
							ReleaseName:               "fake",
							SupportsOperatorUninstall: true,
						},
					},
				}
			})
			defer registry.ResetGetComponentsFn()

			k8sutil.GetCoreV1Func = common.MockGetCoreV1()
			k8sutil.GetDynamicClientFunc = common.MockDynamicClient()
			defer func() {
				k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
				k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
			}()

			reconciler := newVerrazzanoReconciler(c)
			result, err := reconciler.reconcileUninstall(vzlog.DefaultLogger(), vzcr)
			asserts.NoError(err)
			asserts.Equal(true, result.Requeue)
			asserts.NotEqual(time.Duration(0), result.RequeueAfter)

			// reconcile a second time with isInstalled returning false
			registry.OverrideGetComponentsFn(func() []spi.Component {
				return []spi.Component{
					fakeComponent{
						HelmComponent: helm2.HelmComponent{
							ReleaseName:               "fake",
							SupportsOperatorUninstall: true,
						},
						isInstalledFunc: func(ctx spi.ComponentContext) (bool, error) {
							return false, nil
						},
					},
				}
			})
			result, err = reconciler.reconcileUninstall(vzlog.DefaultLogger(), vzcr)
			asserts.NoError(err)
			asserts.Equal(false, result.Requeue)
			asserts.Equal(time.Duration(0), result.RequeueAfter)
			err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, vzcr)
			asserts.NoError(err)
			asserts.NotZero(len(vzcr.Status.Components), "Status.Components len should not be zero")
			asserts.Equal("Uninstalled", string(vzcr.Status.Components["fake"].State), "Invalid component state")
			asserts.NotZero(len(UninstallTrackerMap), "UninstallTrackerMap should have no entries")

			// assert the MC secrets have been deleted
			for _, s := range test.secrets {
				newSecret := &corev1.Secret{}
				err = c.Get(context.TODO(), types.NamespacedName{Namespace: s.Namespace, Name: s.Name}, newSecret)
				if test.managed {
					asserts.True(errors.IsNotFound(err), fmt.Sprintf("Secret %s should not exist", s.Name))
				} else {
					asserts.NoError(err, fmt.Sprintf("Secret %s should exist", s.Name))
				}
			}
			// Assert that MC namespace exists if there is a project
			ns := corev1.Namespace{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: vzconst.VerrazzanoMultiClusterNamespace}, &ns)
			if test.createProject {
				asserts.NoError(err, fmt.Sprintf("Namespace %s should exist since it has projects", ns.Name))
				assertProjectNamespaces(c, asserts)
				assertProjectRoleBindings(c, asserts)
			} else {
				asserts.True(errors.IsNotFound(err), fmt.Sprintf("Namespace %s should not exist since there are no projects", ns.Name))
			}
		})
	}
}

// Build a fake client and load it with the test Kubernetes resources
func buildFakeClientAndObjects(createMCNamespace bool, createProject bool, secrets []corev1.Secret) (client.Client, *vzapi.Verrazzano) {
	namespace := "verrazzano"
	name := "test"
	var verrazzanoToUse vzapi.Verrazzano
	labels := map[string]string{}

	vzcr := &vzapi.Verrazzano{
		ObjectMeta: createObjectMeta(namespace, name, []string{finalizerName}),
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{},
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
				return statusMap
			}(),
		},
	}

	// Add core resources
	cb := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		vzcr,
		rbac.NewServiceAccount(namespace, name, []string{}, labels),
		rbac.NewClusterRoleBinding(&verrazzanoToUse, name, getInstallNamespace(), buildServiceAccountName(name)),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: keycloak.ComponentNamespace,
				Name:      keycloak.ComponentName,
				Labels:    map[string]string{"app": keycloak.ComponentName},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		})

	// Add MC related resources
	objects := []client.Object{}
	for i := range secrets {
		objects = append(objects, &secrets[i])
	}
	if createMCNamespace {
		objects = append(objects,
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: vzconst.VerrazzanoMultiClusterNamespace}})
	}
	if createProject {
		proj := vzappclusters.VerrazzanoProject{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: vzconst.VerrazzanoMultiClusterNamespace},
			Spec: vzappclusters.VerrazzanoProjectSpec{
				Template: vzappclusters.ProjectTemplate{
					Namespaces: []vzappclusters.NamespaceTemplate{
						{Metadata: metav1.ObjectMeta{Name: "projns1"}},
						{Metadata: metav1.ObjectMeta{Name: "projns2"}},
					},
				},
			},
		}
		projns1 := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "projns1"}}
		projns2 := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "projns2"}}
		objects = append(objects, &proj, &projns1, &projns2)
		objects = addFakeProjectRoleBindings(objects)
	}
	cb.WithObjects(objects...)

	return cb.Build(), vzcr
}

func addFakeProjectRoleBindings(objects []client.Object) []client.Object {
	mcClusterRoleRef := rbacv1.RoleRef{
		Name:     "verrazzano-managed-cluster",
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
	}
	otherClusterRoleRef := rbacv1.RoleRef{
		Name:     "somerole",
		Kind:     "ClusterRole",
		APIGroup: rbacv1.GroupName,
	}
	mcRolebinding1 := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster1", Namespace: "projns1"},
		RoleRef:    mcClusterRoleRef,
	}
	mcRolebinding2 := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster2", Namespace: "projns1"},
		RoleRef:    mcClusterRoleRef,
	}
	mcRolebinding3 := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster1", Namespace: "projns2"},
		RoleRef:    mcClusterRoleRef,
	}
	otherRolebinding := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "somerb", Namespace: "projns2"},
		RoleRef:    otherClusterRoleRef,
	}
	clusterRoleManaged := rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano-managed-cluster"}}
	clusterRoleOther := rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "somerole"}}
	return append(objects, &clusterRoleManaged, &clusterRoleOther, &mcRolebinding1, &mcRolebinding2, &mcRolebinding3, &otherRolebinding)
}

func assertProjectNamespaces(c client.Client, asserts *assert.Assertions) {
	ns := corev1.Namespace{}
	for _, nsName := range []string{"projns1", "projns2"} {
		err := c.Get(context.TODO(), types.NamespacedName{Name: nsName}, &ns)
		asserts.NoError(err, "Project namespace %s should exist", nsName)
	}
}

func assertProjectRoleBindings(c client.Client, asserts *assert.Assertions) {
	rblist := rbacv1.RoleBindingList{}
	c.List(context.TODO(), &rblist)
	rb := rbacv1.RoleBinding{}
	roleBindingsDeleted := []types.NamespacedName{
		{Name: "cluster1", Namespace: "projns1"},
		{Name: "cluster2", Namespace: "projns1"},
		{Name: "cluster1", Namespace: "projns2"},
	}
	roleBindingsExist := []types.NamespacedName{
		{Name: "somerb", Namespace: "projns2"},
	}
	for _, rbDeleted := range roleBindingsDeleted {
		err := c.Get(context.TODO(), rbDeleted, &rb)
		asserts.Error(err, "Role cluster1 should have been deleted")
	}
	for _, rbExists := range roleBindingsExist {
		err := c.Get(context.TODO(), rbExists, &rb)
		asserts.NoError(err, "RoleBinding %s/%s should still exist", rbExists.Namespace, rbExists.Name)
	}
}

// TestDeleteNamespaces tests the deleteNamespaces method for the following use case
// GIVEN a request to deleteNamespaces
// WHEN deleteNamespaces is called
// THEN ensure all the component and shared namespaces are deleted
func TestDeleteNamespaces(t *testing.T) {
	asserts := assert.New(t)

	const fakeNS = "foo"
	nameSpaces := []client.Object{}
	names := []string{
		fakeNS,
		appoper.ComponentNamespace,
		authproxy.ComponentNamespace,
		certmanager.ComponentNamespace,
		coherence.ComponentNamespace,
		console.ComponentNamespace,
		externaldns.ComponentNamespace,
		fluentd.ComponentNamespace,
		grafana.ComponentNamespace,
		istio.IstioNamespace,
		jaegeroperator.ComponentNamespace,
		keycloak.ComponentNamespace,
		kiali.ComponentNamespace,
		mysql.ComponentNamespace,
		nginx.ComponentNamespace,
		oam.ComponentNamespace,
		opensearch.ComponentNamespace,
		opensearchdashboards.ComponentNamespace,
		rancher.ComponentNamespace,
		velero.ComponentNamespace,
		verrazzano.ComponentNamespace,
		vmo.ComponentNamespace,
		weblogic.ComponentNamespace,

		// Shared
		vzconst.VerrazzanoMonitoringNamespace,
		constants.CertManagerNamespace,
		constants.VerrazzanoSystemNamespace,
		vzconst.KeycloakNamespace,
		monitoringNamespace,
	}
	// Remove dups since adding objects to the fake client with fail on duplicates
	nsSet := make(map[string]bool)
	for i := range names {
		nsSet[names[i]] = true
	}
	for n := range nsSet {
		nameSpaces = append(nameSpaces, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: n},
		})
	}

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	_ = v1alpha1.AddToScheme(k8scheme.Scheme)
	_ = vzappclusters.AddToScheme(k8scheme.Scheme)

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(nameSpaces...).Build()

	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.deleteNamespaces(vzlog.DefaultLogger(), false)

	// Validate the results
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	for _, n := range names {
		ns := &corev1.Namespace{}
		err = c.Get(context.TODO(), types.NamespacedName{Name: n}, ns)
		if err == nil {
			asserts.Equal(ns.Name, fakeNS, fmt.Sprintf("Namespace %s should exist", fakeNS))
		} else {
			asserts.True(errors.IsNotFound(err), fmt.Sprintf("Namespace %s should not exist", n))
		}
	}
}

// TestReconcileUninstall2 tests reconcileUninstall with negative cases
func TestReconcileUninstall2(t *testing.T) {
	type args struct {
		log vzlog.VerrazzanoLogger
		cr  *vzapi.Verrazzano
	}
	helmOverrideNotFound := func() {
		helm.SetCmdRunner(vzos.GenericTestRunner{
			StdOut: []byte(""),
			StdErr: []byte("not found"),
			Err:    fmt.Errorf(unExpectedError),
		})
	}
	helmOverrideNoError := func() {
		helm.SetCmdRunner(vzos.GenericTestRunner{
			StdOut: []byte(""),
			StdErr: []byte(""),
			Err:    nil,
		})
	}
	defer helm.SetDefaultRunner()
	config.TestProfilesDir = relativeProfilesDir

	k8sutil.GetCoreV1Func = common.MockGetCoreV1()
	k8sutil.GetDynamicClientFunc = common.MockDynamicClient()
	defer func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
	}()

	getMockWithError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getDeletionMock := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.Not(nil), gomock.Any()).Return(nil)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	tests := []struct {
		name             string
		args             args
		getClientFunc    func() client.Client
		helmOverrideFunc func()
		want             controllerruntime.Result
		wantErr          bool
	}{
		// GIVEN VZ reconciler object
		// WHEN reconcileUninstall is called
		// THEN error is returned with empty result for reconcile revocation if resource deletion gets failed
		{
			"TestReconcileUninstall2  when deletion of multi-cluster related resources get failed",
			args{vzlog.DefaultLogger(), &vzapi.Verrazzano{}},
			getMockWithError,
			helmOverrideNotFound,
			controllerruntime.Result{},
			true,
		},
		// GIVEN VZ reconciler object
		// WHEN reconcileUninstall is called
		// THEN error is returned with empty result for reconcile revocation if component is already installed
		{
			"TestReconcileUninstall2  when component is already installed",
			args{vzlog.DefaultLogger(), &vzapi.Verrazzano{ObjectMeta: metav1.ObjectMeta{Name: "testName", Namespace: "testNs"}}},
			getDeletionMock,
			helmOverrideNoError,
			controllerruntime.Result{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reconciler Reconciler
			if tt.helmOverrideFunc != nil {
				tt.helmOverrideFunc()
			}
			if tt.getClientFunc != nil {
				reconciler = newVerrazzanoReconciler(tt.getClientFunc())
			} else {
				reconciler = newVerrazzanoReconciler(nil)
			}
			got, err := reconciler.reconcileUninstall(tt.args.log, tt.args.cr)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileUninstall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reconcileUninstall() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestReconcilerDeleteMCResources tests deleteMCResources
func TestReconcilerDeleteMCResources(t *testing.T) {
	projects := vzappclusters.VerrazzanoProjectList{
		Items: []vzappclusters.VerrazzanoProject{
			{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "testsNS",
					Name:      "testName",
				},
				Spec: vzappclusters.VerrazzanoProjectSpec{
					Template: vzappclusters.ProjectTemplate{
						Namespaces: []vzappclusters.NamespaceTemplate{
							{Metadata: metav1.ObjectMeta{
								Name:      "test1",
								Namespace: "namespace1",
							}},
						},
					},
				},
			},
		},
	}
	managedClusters := []v1alpha1.VerrazzanoManagedCluster{
		{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "testsNS",
				Name:      "testName",
			},
			Spec: v1alpha1.VerrazzanoManagedClusterSpec{
				ServiceAccount: "serviceAccount",
			},
		},
	}
	getMockWithError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.Not(nil), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getMockDeletionError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&vzappclusters.VerrazzanoProjectList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*vzappclusters.VerrazzanoProjectList).Items = projects.Items
			return []interface{}{nil}
		})
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&rbacv1.RoleBindingList{}), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getMockListError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&vzappclusters.VerrazzanoProjectList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*vzappclusters.VerrazzanoProjectList).Items = projects.Items
			return []interface{}{nil}
		})
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&rbacv1.RoleBindingList{}), gomock.Any()).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedClusterList{}), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getMockSAError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&vzappclusters.VerrazzanoProjectList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*vzappclusters.VerrazzanoProjectList).Items = projects.Items
			return []interface{}{nil}
		})
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&rbacv1.RoleBindingList{}), gomock.Any()).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedClusterList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*v1alpha1.VerrazzanoManagedClusterList).Items = managedClusters
			return []interface{}{nil}
		})
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getMockVMCError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&vzappclusters.VerrazzanoProjectList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*vzappclusters.VerrazzanoProjectList).Items = projects.Items
			return []interface{}{nil}
		})
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&rbacv1.RoleBindingList{}), gomock.Any()).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedClusterList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*v1alpha1.VerrazzanoManagedClusterList).Items = managedClusters
			return []interface{}{nil}
		})
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{})).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}

	getMockNSError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&vzappclusters.VerrazzanoProjectList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			return []interface{}{nil}
		})
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedClusterList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*v1alpha1.VerrazzanoManagedClusterList).Items = managedClusters
			return []interface{}{nil}
		})
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Namespace{}), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getMockSecretError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&vzappclusters.VerrazzanoProjectList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			return []interface{}{nil}
		})
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedClusterList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*v1alpha1.VerrazzanoManagedClusterList).Items = managedClusters
			return []interface{}{nil}
		})
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Namespace{}), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: vzconst.VerrazzanoSystemNamespace, Name: vzconst.MCRegistrationSecret}}), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getMockESSecretError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&vzappclusters.VerrazzanoProjectList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			return []interface{}{nil}
		})
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedClusterList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*v1alpha1.VerrazzanoManagedClusterList).Items = managedClusters
			return []interface{}{nil}
		})
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Namespace{}), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: vzconst.VerrazzanoSystemNamespace, Name: vzconst.MCRegistrationSecret}}), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: vzconst.VerrazzanoSystemNamespace, Name: mcElasticSearchScrt}}), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getMockMCSecretError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&vzappclusters.VerrazzanoProjectList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			return []interface{}{nil}
		})
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedClusterList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*v1alpha1.VerrazzanoManagedClusterList).Items = managedClusters
			return []interface{}{nil}
		})
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Namespace{}), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: vzconst.VerrazzanoSystemNamespace, Name: vzconst.MCRegistrationSecret}}), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: vzconst.VerrazzanoSystemNamespace, Name: mcElasticSearchScrt}}), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: vzconst.VerrazzanoSystemNamespace, Name: vzconst.MCAgentSecret}}), gomock.Any()).Return(fmt.Errorf(unExpectedError))
		return mockClient
	}
	getMockNoError := func() client.Client {
		mocker := gomock.NewController(t)
		mockClient := mocks.NewMockClient(mocker)
		mockClient.EXPECT().Get(context.TODO(), gomock.Not(nil), gomock.AssignableToTypeOf(&corev1.Secret{})).Return(nil)
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&vzappclusters.VerrazzanoProjectList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			return []interface{}{nil}
		})
		mockClient.EXPECT().List(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedClusterList{}), gomock.Any()).DoAndReturn(func(ctx context.Context, list client.ObjectList, opts *client.ListOptions) []interface{} {
			list.(*v1alpha1.VerrazzanoManagedClusterList).Items = managedClusters
			return []interface{}{nil}
		})
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.ServiceAccount{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&v1alpha1.VerrazzanoManagedCluster{})).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Namespace{}), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: vzconst.VerrazzanoSystemNamespace, Name: vzconst.MCRegistrationSecret}}), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: vzconst.VerrazzanoSystemNamespace, Name: mcElasticSearchScrt}}), gomock.Any()).Return(nil)
		mockClient.EXPECT().Delete(context.TODO(), gomock.AssignableToTypeOf(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: vzconst.VerrazzanoSystemNamespace, Name: vzconst.MCAgentSecret}}), gomock.Any()).Return(nil)
		return mockClient
	}
	tests := []struct {
		name          string
		ctx           spi.ComponentContext
		getClientFunc func() client.Client
		wantErr       bool
	}{
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN error is returned if call to list multi-cluster resource gets failed
		{
			"TestReconcilerDeleteMCResources when call to list multi-cluster resource gets failed",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockWithError,
			true,
		},
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN error is returned if call to delete multi-cluster role-bindings gets failed
		{
			"TestReconcilerDeleteMCResources when deleteManagedClusterRoleBindings gets failed",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockDeletionError,
			true,
		},
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN error is returned if call to list  VerrazzanoManagedClusterList gets failed
		{
			"TestReconcilerDeleteMCResources when getting of list of VerrazzanoManagedClusterList failed",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockListError,
			true,
		},
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN error is returned if call to delete service account gets failed
		{
			"TestReconcilerDeleteMCResources when service account deletion failed",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockSAError,
			true,
		},
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN error is returned if call to delete Verrazzano managed cluster gets failed
		{
			"TestReconcilerDeleteMCResources when Verrazzano managed cluster deletion failed",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockVMCError,
			true,
		},
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN error is returned if call to delete namespace gets failed
		{
			"TestReconcilerDeleteMCResources when namespace deletion failed",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockNSError,
			true,
		},
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN error is returned if call to delete MC registration secret gets failed
		{
			"TestReconcilerDeleteMCResources when MC registration secret deletion failed",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockSecretError,
			true,
		},
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN error is returned if call to delete ES secret gets failed
		{
			"TestReconcilerDeleteMCResources when ES secret deletion failed",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockESSecretError,
			true,
		},
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN error is returned if call to delete MC agent secret gets failed
		{
			"TestReconcilerDeleteMCResources when MC agent secret failed",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockMCSecretError,
			true,
		},
		// GIVEN VZ reconciler
		// WHEN deleteMCResources is called
		// THEN no error is returned if there is no error
		{
			"TestReconcilerDeleteMCResources when no error",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, true),
			getMockNoError,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reconciler Reconciler
			if tt.getClientFunc != nil {
				reconciler = newVerrazzanoReconciler(tt.getClientFunc())
			} else {
				reconciler = newVerrazzanoReconciler(nil)
			}
			if err := reconciler.deleteMCResources(tt.ctx); (err != nil) != tt.wantErr {
				t.Errorf("deleteMCResources() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
