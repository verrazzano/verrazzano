// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/console"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	jaegeroperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/jaeger/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearchdashboards"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/velero"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"

	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"

	helm2 "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/stretchr/testify/assert"
	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/rbac"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
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

	config.TestProfilesDir = "../../manifests/profiles/v1alpha1"
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
	_ = clustersv1alpha1.AddToScheme(k8scheme.Scheme)
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

	config.TestProfilesDir = "../../manifests/profiles/v1alpha1"
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
	_ = clustersv1alpha1.AddToScheme(k8scheme.Scheme)
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

			config.TestProfilesDir = "../../manifests/profiles/v1alpha1"
			defer func() { config.TestProfilesDir = "" }()

			defer config.Set(config.Get())
			config.Set(config.OperatorConfig{VersionCheckEnabled: false})

			_ = vzapi.AddToScheme(k8scheme.Scheme)
			_ = clustersv1alpha1.AddToScheme(k8scheme.Scheme)
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
	_ = clustersv1alpha1.AddToScheme(k8scheme.Scheme)
	_ = vzappclusters.AddToScheme(k8scheme.Scheme)

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(nameSpaces...).Build()

	reconciler := newVerrazzanoReconciler(c)
	result, err := reconciler.deleteNamespaces(vzlog.DefaultLogger())

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
