// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"

	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	nodePort                    = installv1beta1.NodePort
	compName                    = "istio"
	ExternalIPArg               = "gateways.istio-ingressgateway.externalIPs"
	specServiceJSONPath         = "spec.components.ingressGateways.0.k8s.service"
	externalIPJsonPathSuffixAt0 = "externalIPs.0"
	externalIPJsonPathSuffix    = "externalIPs"
	typeJSONPathSuffix          = "type"
	externalIPJsonPath          = specServiceJSONPath + "." + externalIPJsonPathSuffix
	externalIPJsonPathAt0       = specServiceJSONPath + "." + externalIPJsonPathSuffixAt0
	validIP                     = "0.0.0.0"
	invalidIP                   = "0.0.0"
	formatError                 = "Must be a proper IP address format"
)

// TestCheckExternalIPsArgs tests CheckExternalIPsArgs
// GIVEN a v1alpha1 VZ CR with ExternalIP IstioOverrides
// WHEN the override key is not found or the IP is invalid
// THEN return an error, nil otherwise
func TestCheckExternalIPsArgs(t *testing.T) {
	asserts := assert.New(t)
	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	vz := getVZWithIstioOverride(validIP)
	err := CheckExternalIPsArgs(vz.Spec.Components.Istio.IstioInstallArgs, vz.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPathAt0, compName, vz.Namespace)
	asserts.NoError(err)
	vz = getVZWithIstioOverride(invalidIP)
	err = CheckExternalIPsArgs(vz.Spec.Components.Istio.IstioInstallArgs, vz.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPathAt0, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), formatError)

	vz = getVZWithIstioOverride("")
	err = CheckExternalIPsArgs(vz.Spec.Components.Istio.IstioInstallArgs, vz.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPathAt0, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), "not found for component")
}

// TestCheckExternalIPsOverridesArgs tests CheckExternalIPsOverridesArgs
// GIVEN a v1beta1 VZ CR with ExternalIP IstioOverrides
// WHEN the IP is not valid
// THEN return an error, nil otherwise
func TestCheckExternalIPsOverridesArgs(t *testing.T) {
	asserts := assert.New(t)
	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZWithIstioOverride(validIP)
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)

	vz = getv1beta1VZWithIstioOverride(invalidIP)
	err = CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), formatError)
}

// TestCheckExternalIPsOverridesArgsWithPaths tests CheckExternalIPsOverridesArgsWithPaths
// GIVEN a v1beta1 VZ CR with ExternalIP IstioOverrides
// WHEN the IP is not valid
// THEN return an error, nil otherwise
func TestCheckExternalIPsOverridesArgsWithPaths(t *testing.T) {
	asserts := assert.New(t)
	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZWithIstioOverride(validIP)
	err := CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.NoError(err)

	vz = getv1beta1VZWithIstioOverride(invalidIP)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), formatError)
}

func TestIfConfigMap(t *testing.T) {
	asserts := assert.New(t)
	var validIp = "1.1.1.1"
	var invalidIp = "1.1."

	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(createTestConfigMap(validIp)).Build(), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()
	vz := getVZWithConfigMapOverride()
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)

	// Passes an invalid IP
	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(createTestConfigMap(invalidIp)).Build(), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()
	vz = getVZWithConfigMapOverride()
	err = CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)

	//Test CheckExternalIPsArgs with ConfigMap
	getControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(createTestConfigMap(invalidIP)).Build(), nil
	}
	defer func() { getControllerRuntimeClient = validators.GetClient }()
	v1alpha1VZ := getv1alpha1VZWithConfigMapOverride()
	err = CheckExternalIPsArgs(v1alpha1VZ.Spec.Components.Istio.IstioInstallArgs, v1alpha1VZ.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
}

// getIstioOverride returns an Istio override in json format
func getIstioOverride(externalIP string) []byte {
	override := fmt.Sprintf(`spec:
  components:
    ingressGateways:
      - name: istio-ingressgateway
        k8s:
          service:
            type: NodePort
            externalIPs:    
            - %s`, externalIP)

	json, _ := yaml.ToJSON([]byte(override))
	return json
}

// getVZWithIstioOverride returns v1aplha1 vz CR with Istio Component overrides
func getVZWithIstioOverride(externalIP string) vzapi.Verrazzano {
	if len(externalIP) == 0 {
		return vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Istio: &vzapi.IstioComponent{},
				},
			},
		}
	}
	vz := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: getIstioOverride(externalIP),
								},
							},
						},
					},
				},
			},
		},
	}
	return vz
}

// getv1beta1VZWithIstioOverride returns v1beta1 vz CR with Istio Component overrides
func getv1beta1VZWithIstioOverride(externalIP string) installv1beta1.Verrazzano {
	vz := installv1beta1.Verrazzano{
		Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Istio: &installv1beta1.IstioComponent{
					InstallOverrides: installv1beta1.InstallOverrides{
						ValueOverrides: []installv1beta1.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: getIstioOverride(externalIP),
								},
							},
						},
					},
				},
			},
		},
	}
	return vz
}

// getv1alpha1VZWithConfigMapOverride returns v1alpha1 vz with istio component using a configmap override
func getv1alpha1VZWithConfigMapOverride() vzapi.Verrazzano {
	vz := vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{
								ConfigMapRef: &corev1.ConfigMapKeySelector{
									Key: "configMapKey",
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "testCMName",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return vz
}

// getVZWithConfigMapOverride returns vz CR with istio component using a configMap override
func getVZWithConfigMapOverride() installv1beta1.Verrazzano {
	vz := installv1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Istio: &installv1beta1.IstioComponent{
					InstallOverrides: installv1beta1.InstallOverrides{
						ValueOverrides: []installv1beta1.Overrides{
							{
								ConfigMapRef: &corev1.ConfigMapKeySelector{
									Key: "configMapKey",
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "testCMName",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return vz
}

func createTestConfigMap(testIP string) *corev1.ConfigMap {
	data := make(map[string]string)
	data["configMapKey"] = createConfigMapData(testIP)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "testCMName",
		},
		Data: data,
	}
}

func createConfigMapData(testIP string) string {
	data := fmt.Sprintf(`spec:
  components:
    ingressGateways:
    - k8s:
        service:
          externalIPs:
          - 0.0.0.0
          - ` + testIP + `
          type: NodePort
      name: istio-ingressgateway`)
	return data
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	return scheme
}
