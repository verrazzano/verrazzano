// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"testing"
	"time"

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
	"github.com/verrazzano/verrazzano/pkg/helm"
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

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	config.TestProfilesDir = "../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

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
		// Admin cluster test
		{
			name:              "admin-cluster",
			managed:           false,
			createMCNamespace: true,
			secrets: []corev1.Secret{
				{ObjectMeta: metav1.ObjectMeta{Name: vzconst.MCRegistrationSecret, Namespace: vzconst.VerrazzanoSystemNamespace}},
				{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano-cluster-elasticsearch", Namespace: vzconst.VerrazzanoSystemNamespace}},
			},
		},
		// Admin cluster test with project
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
				{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano-cluster-elasticsearch", Namespace: vzconst.VerrazzanoSystemNamespace}},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initUnitTesing()
			namespace := "verrazzano"
			name := "test"
			var verrazzanoToUse vzapi.Verrazzano
			labels := map[string]string{}

			config.SetDefaultBomFilePath(unitTestBomFile)
			asserts := assert.New(t)

			defer config.Set(config.Get())
			config.Set(config.OperatorConfig{VersionCheckEnabled: false})

			config.TestProfilesDir = "../../manifests/profiles"
			defer func() { config.TestProfilesDir = "" }()

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
			for i, _ := range test.secrets {
				objects = append(objects, &test.secrets[i])
			}
			if test.createMCNamespace {
				objects = append(objects,
					&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: vzconst.VerrazzanoMultiClusterNamespace}})
			}
			if test.createProject {
				objects = append(objects,
					&vzappclusters.VerrazzanoProject{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: vzconst.VerrazzanoMultiClusterNamespace}})
			}
			cb.WithObjects(objects...)

			c := cb.Build()

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

			// assert the MC secrets have been deleted
			for _, s := range test.secrets {
				err = c.Get(context.TODO(), types.NamespacedName{Namespace: s.Namespace, Name: s.Name}, &s)
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
			} else {
				asserts.True(errors.IsNotFound(err), fmt.Sprintf("Namespace %s should not exist since there are no projects", ns.Name))
			}
		})
	}
}
