// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	appsv1Cli "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1Cli "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

// MockGetCoreV1 mocks GetCoreV1Client function
func MockGetCoreV1(objects ...runtime.Object) func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
	return func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(objects...).CoreV1(), nil
	}
}

// MockGetCoreV1WithNamespace mocks GetCoreV1Client function with a Get for a verrazzano managed namespace
func MockGetCoreV1WithNamespace(namespace string, objects ...runtime.Object) func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
	return func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		var newObjects = []runtime.Object{
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						constants.LabelVerrazzanoNamespace: namespace,
					},
				},
			},
		}
		if objects != nil {
			newObjects = append(newObjects, objects...)
		}
		return k8sfake.NewSimpleClientset(newObjects...).CoreV1(), nil
	}
}

// MockGetAppsV1 mocks GetAppsV1Client function
func MockGetAppsV1(objects ...runtime.Object) func(_ ...vzlog.VerrazzanoLogger) (appsv1Cli.AppsV1Interface, error) {
	return func(_ ...vzlog.VerrazzanoLogger) (appsv1Cli.AppsV1Interface, error) {
		return k8sfake.NewSimpleClientset(objects...).AppsV1(), nil
	}
}

// MockDynamicClient mocks GetDynamicClient function
func MockDynamicClient(objects ...runtime.Object) func() (dynamic.Interface, error) {
	return func() (dynamic.Interface, error) {
		return dynfake.NewSimpleDynamicClient(runtime.NewScheme(), objects...), nil
	}
}

func MkSvc(ns, name string) *corev1.Service {
	svc := &corev1.Service{}
	svc.Namespace = ns
	svc.Name = name
	return svc
}

func MkDep(ns, name string) *appsv1.Deployment {
	dep := &appsv1.Deployment{}
	dep.Namespace = ns
	dep.Name = name
	return dep
}

type ValidateInstallTest struct {
	Name       string
	Vz         *vzapi.Verrazzano
	Corev1Cli  func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error)
	Appsv1Cli  func(_ ...vzlog.VerrazzanoLogger) (appsv1Cli.AppsV1Interface, error)
	DynamicCli func() (dynamic.Interface, error)
	WantErr    string
}

// RunValidateInstallTest runs ValidateInstallTests
func RunValidateInstallTest(t *testing.T, newComp func() spi.Component, tests ...ValidateInstallTest) {
	resetGetClients := func() {
		k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client
		k8sutil.GetAppsV1Func = k8sutil.GetAppsV1Client
		k8sutil.GetDynamicClientFunc = k8sutil.GetDynamicClient
	}
	defer resetGetClients()
	assertErr := func(wantErr string, err error) {
		if wantErr == "" {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), wantErr)
		}
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			k8sutil.GetCoreV1Func = tt.Corev1Cli
			k8sutil.GetAppsV1Func = tt.Appsv1Cli
			k8sutil.GetDynamicClientFunc = tt.DynamicCli
			c := newComp()
			assertErr(tt.WantErr, c.ValidateInstall(tt.Vz))
			vzV1Beta1 := &v1beta1.Verrazzano{}
			err := tt.Vz.ConvertTo(vzV1Beta1)
			assert.NoError(t, err)
			assertErr(tt.WantErr, c.ValidateInstallV1Beta1(vzV1Beta1))
			resetGetClients()
		})
	}
}

// BoolPtr returns a pointer to the given bool value.
func BoolPtr(value bool) *bool {
	return &value
}

// Int32Ptr returns a pointer to the given int32 value
func Int32Ptr(v int32) *int32 {
	return &v
}
