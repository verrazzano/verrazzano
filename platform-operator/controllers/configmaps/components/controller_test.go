// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package components

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	testBomFilePath = "../../verrazzano/testdata/test_bom.json"
)

// TestConfigMapReconciler tests ComponentConfigMapReconciler method for a correct and incorrect configmap
func TestConfigMapReconciler(t *testing.T) {
	asserts := assert.New(t)

	tests := []struct {
		name        string
		cm          corev1.ConfigMap
		err         error
		returnError bool
		requeue     bool
	}{
		{
			name: "successful installation",
			cm: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-component",
					Namespace: constants.VerrazzanoInstallNamespace,
					Annotations: map[string]string{
						constants.VerrazzanoDevComponentAnnotationName: "test-component",
					},
				},
				Data: map[string]string{
					chartPathKey:          "test-component",
					componentNamespaceKey: constants.VerrazzanoSystemNamespace,
					overridesKey:          "overrideKey: overrideVal",
				},
			},
			returnError: false,
			requeue:     false,
		},
		{
			name: "configmap in wrong namespace",
			cm: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-component",
					Namespace: constants.VerrazzanoSystemNamespace,
					Annotations: map[string]string{
						constants.VerrazzanoDevComponentAnnotationName: "test-component",
					},
				},
				Data: map[string]string{
					chartPathKey:          "test-component",
					componentNamespaceKey: constants.VerrazzanoSystemNamespace,
					overridesKey:          "overrideKey: overrideVal",
				},
			},
			returnError: true,
			requeue:     false,
		},
		{
			name: "no chart path",
			cm: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-component",
					Namespace: constants.VerrazzanoInstallNamespace,
					Annotations: map[string]string{
						constants.VerrazzanoDevComponentAnnotationName: "test-component",
					},
				},
				Data: map[string]string{
					componentNamespaceKey: constants.VerrazzanoSystemNamespace,
					overridesKey:          "overrideKey: overrideVal",
				},
			},
			returnError: true,
			requeue:     false,
		},
		{
			name: "no namespace",
			cm: corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-component",
					Namespace: constants.VerrazzanoInstallNamespace,
					Annotations: map[string]string{
						constants.VerrazzanoDevComponentAnnotationName: "test-component",
					},
				},
				Data: map[string]string{
					chartPathKey: "test-component",
					overridesKey: "overrideKey: overrideVal",
				},
			},
			returnError: true,
			requeue:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetDefaultBomFilePath(testBomFilePath)
			helm.SetUpgradeFunc(fakeUpgrade)
			defer helm.SetDefaultUpgradeFunc()
			config.TestProfilesDir = "../../../manifests/profiles"
			defer func() { config.TestProfilesDir = "" }()

			cli := buildFakeClient(tt.cm)

			req := newRequest(tt.cm.Namespace, tt.cm.Name)
			reconciler := newConfigMapReconciler(cli)
			res, err := reconciler.Reconcile(context.TODO(), req)

			if tt.returnError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}

			asserts.Equal(tt.requeue, res.Requeue)

		})
	}
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	return scheme
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) controllerruntime.Request {
	return controllerruntime.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name}}
}

// newConfigMapReconciler creates a new reconciler for testing
func newConfigMapReconciler(c client.Client) ComponentConfigMapReconciler {
	scheme := newScheme()
	reconciler := ComponentConfigMapReconciler{
		Client: c,
		Scheme: scheme,
	}
	return reconciler
}

func buildFakeClient(cm corev1.ConfigMap) client.Client {
	vz := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vz",
			Namespace: constants.VerrazzanoInstallNamespace,
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.GlobalImagePullSecName,
			Namespace: "default",
		},
	}
	return fake.NewClientBuilder().WithObjects(vz, &cm, secret).WithScheme(newScheme()).Build()
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, _ bool, _ bool, overrides []helmcli.HelmOverrides) (stdout []byte, stderr []byte, err error) {
	if releaseName != "test-component" {
		return []byte("error"), []byte(""), fmt.Errorf("Incorrect  releaseName, expecting test-component, got %s", releaseName)
	}
	if chartDir != "/verrazzano/platform-operator/thirdparty/charts/test-component" {
		return []byte("error"), []byte(""), fmt.Errorf("Incorrect  releaseName, expecting test-component, got %s", chartDir)
	}
	if namespace != constants.VerrazzanoSystemNamespace {
		return []byte("error"), []byte(""), fmt.Errorf("Incorrect release namespace, expecting verrazzano-system, got %s", namespace)
	}
	if len(overrides) != 2 {
		return []byte("error"), []byte(""), fmt.Errorf("Incorrect number of overrides, expecting 2, got %d", len(overrides))
	}
	return []byte("success"), []byte(""), nil
}
