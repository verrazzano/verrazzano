// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	nodePort                 = installv1beta1.NodePort
	compName                 = "istio"
	ExternalIPArg            = "gateways.istio-ingressgateway.externalIPs"
	specServiceJSONPath      = "spec.components.ingressGateways.0.k8s.service"
	externalIPJsonPathSuffix = "externalIPs"
	typeJSONPathSuffix       = "type"
	externalIPJsonPath       = specServiceJSONPath + "." + externalIPJsonPathSuffix
	validIP                  = "0.0.0.0"
	invalidIP                = "0.0.0"
	formatError              = "Must be a proper IP address format"
	configMap                = "configMap"
	secret                   = "secret"
	values                   = "values"
)

// TestCheckExternalIPsArgs tests CheckExternalIPsArgs
// GIVEN a v1alpha1 VZ CR with ExternalIP IstioOverrides
// WHEN the override key is not found or the IP is invalid
// THEN return an error, nil otherwise
func TestCheckExternalIPsArgs(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClient()
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1alpha1VZ()
	createv1alpha1VZOverrides(vz, "", "", values, validIP)
	err := CheckExternalIPsArgs(vz.Spec.Components.Istio.IstioInstallArgs, vz.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)
	createv1alpha1VZOverrides(vz, "", "", values, invalidIP)
	err = CheckExternalIPsArgs(vz.Spec.Components.Istio.IstioInstallArgs, vz.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), formatError)
	createv1alpha1VZOverrides(vz, "", "", values, "")
	err = CheckExternalIPsArgs(vz.Spec.Components.Istio.IstioInstallArgs, vz.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), "invalid data type")
}

// TestCheckExternalIPsOverridesArgs tests CheckExternalIPsOverridesArgs
// GIVEN a v1beta1 VZ CR with ExternalIP IstioOverrides
// WHEN the IP is not valid
// THEN return an error, nil otherwise
func TestCheckExternalIPsOverridesArgs(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClient()
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, "", "", values, validIP)
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)
	createv1beta1VZOverrides(vz, "", "", values, invalidIP)
	err = CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), formatError)
	createv1beta1VZOverrides(vz, "", "", values, "")
	err = CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), "invalid data type")
}

// TestCheckExternalIPsOverridesArgsWithPaths tests CheckExternalIPsOverridesArgsWithPaths
// GIVEN a v1beta1 VZ CR with ExternalIP IstioOverrides
// WHEN the IP is not valid
// THEN return an error, nil otherwise
func TestCheckExternalIPsOverridesArgsWithPaths(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClient()
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, "", "", values, validIP)
	err := CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.NoError(err)
	createv1beta1VZOverrides(vz, "", "", values, invalidIP)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), formatError)
	createv1beta1VZOverrides(vz, "", "", values, "")
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.Error(err)
	asserts.Contains(err.Error(), "invalid data type")
}

func TestValidIPWithConfigMapOverride(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClientWithConfigMap(createTestConfigMap(true, validIP, validIP))
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, configMap, "", "", validIP)
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.NoError(err)

	// Test CheckExternalIPsArgs uses a v1alpha1 vz resource
	v1alpha1VZ := createv1alpha1VZOverrides(getv1alpha1VZ(), configMap, "", "", validIP)
	err = CheckExternalIPsArgs(v1alpha1VZ.Spec.Components.Istio.IstioInstallArgs, v1alpha1VZ.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)
}

func TestInvalidIPWithConfigMapOverride(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClientWithConfigMap(createTestConfigMap(true, invalidIP, invalidIP))
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, configMap, "", "", "")
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.Error(err)

	// Test CheckExternalIPsArgs uses a v1alpha1 vz resource
	v1alpha1VZ := getv1alpha1VZ()
	createv1alpha1VZOverrides(v1alpha1VZ, configMap, "", "", "")
	err = CheckExternalIPsArgs(v1alpha1VZ.Spec.Components.Istio.IstioInstallArgs, v1alpha1VZ.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
}

func TestValidIPWithSecretOverride(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClientWithSecret(createTestSecret(true, validIP, validIP))
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, "", secret, "", "")
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.NoError(err)

	// Test CheckExternalIPsArgs uses a v1alpha1 vz resource
	v1alpha1VZ := getv1alpha1VZ()
	createv1alpha1VZOverrides(v1alpha1VZ, "", secret, "", "")
	err = CheckExternalIPsArgs(v1alpha1VZ.Spec.Components.Istio.IstioInstallArgs, v1alpha1VZ.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)
}

func TestInvalidIPWithSecretOverride(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClientWithSecret(createTestSecret(true, "", ""))
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, "", secret, invalidIP, invalidIP)
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.Error(err)

	// Test CheckExternalIPsArgs uses a v1alpha1 vz resource
	v1alpha1VZ := getv1alpha1VZ()
	createv1alpha1VZOverrides(v1alpha1VZ, "", secret, "", "")
	err = CheckExternalIPsArgs(v1alpha1VZ.Spec.Components.Istio.IstioInstallArgs, v1alpha1VZ.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
}

func TestValidIPWithConfigMapSecretOverride(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClientWithConfigMapAndSecret(
		createTestConfigMap(false, validIP, ""),
		createTestSecret(false, validIP, ""))
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, configMap, secret, "", "")
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.NoError(err)

	// Test CheckExternalIPsArgs uses a v1alpha1 vz resource
	v1alpha1VZ := getv1alpha1VZ()
	createv1alpha1VZOverrides(v1alpha1VZ, configMap, secret, "", "")
	err = CheckExternalIPsArgs(v1alpha1VZ.Spec.Components.Istio.IstioInstallArgs, v1alpha1VZ.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)
}

func TestInvalidIPWithConfigMapSecretOverride(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClientWithConfigMapAndSecret(
		createTestConfigMap(false, invalidIP, ""),
		createTestSecret(false, invalidIP, ""))
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, configMap, secret, "", "")
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.Error(err)

	// Test CheckExternalIPsArgs uses a v1alpha1 vz resource
	v1alpha1VZ := getv1alpha1VZ()
	createv1alpha1VZOverrides(v1alpha1VZ, configMap, secret, "", "")
	err = CheckExternalIPsArgs(v1alpha1VZ.Spec.Components.Istio.IstioInstallArgs, v1alpha1VZ.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
}

func TestValidIPWithConfigMapValue(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClientWithConfigMap(createTestConfigMap(false, validIP, ""))
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, configMap, "", values, validIP)
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.NoError(err)

	// Test CheckExternalIPsArgs uses a v1alpha1 vz resource
	v1alpha1VZ := getv1alpha1VZ()
	createv1alpha1VZOverrides(v1alpha1VZ, configMap, "", values, validIP)
	err = CheckExternalIPsArgs(v1alpha1VZ.Spec.Components.Istio.IstioInstallArgs, v1alpha1VZ.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.NoError(err)

}

func TestInvalidIPWithConfigMapValue(t *testing.T) {
	asserts := assert.New(t)
	createFakeTestClientWithConfigMap(createTestConfigMap(false, invalidIP, ""))
	defer func() { GetControllerRuntimeClient = validators.GetClient }()

	vz := getv1beta1VZ()
	createv1beta1VZOverrides(vz, configMap, "", values, invalidIP)
	err := CheckExternalIPsOverridesArgs(vz.Spec.Components.Istio.ValueOverrides, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)
	err = CheckExternalIPsOverridesArgsWithPaths(vz.Spec.Components.Istio.ValueOverrides, specServiceJSONPath, typeJSONPathSuffix, string(nodePort), externalIPJsonPathSuffix, compName, vz.Namespace)
	asserts.Error(err)

	// Test CheckExternalIPsArgs uses a v1alpha1 vz resource
	v1alpha1VZ := getv1alpha1VZ()
	createv1alpha1VZOverrides(v1alpha1VZ, configMap, "", values, invalidIP)
	err = CheckExternalIPsArgs(v1alpha1VZ.Spec.Components.Istio.IstioInstallArgs, v1alpha1VZ.Spec.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, compName, vz.Namespace)
	asserts.Error(err)

}

// getv1alpha1VZ returns v1beta1 vz CR with Empty ValueOverirides
func getv1alpha1VZ() vzapi.Verrazzano {
	vz := vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{}, {}, {},
						},
					},
				},
			},
		},
	}
	return vz
}

// getv1beta1VZ returns v1beta1 vz CR with Empty ValueOverirides
func getv1beta1VZ() installv1beta1.Verrazzano {
	vz := installv1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
		Spec: installv1beta1.VerrazzanoSpec{
			Components: installv1beta1.ComponentSpec{
				Istio: &installv1beta1.IstioComponent{
					InstallOverrides: installv1beta1.InstallOverrides{
						ValueOverrides: []installv1beta1.Overrides{
							{}, {}, {},
						},
					},
				},
			},
		},
	}
	return vz
}

func createv1beta1VZOverrides(vz installv1beta1.Verrazzano, configMap, secret, values, testIP string) installv1beta1.Verrazzano {
	if configMap != "" {
		vz.Spec.Components.Istio.InstallOverrides.ValueOverrides[0].ConfigMapRef = createTestConfigMapKeySelector()
	}
	if secret != "" {
		vz.Spec.Components.Istio.InstallOverrides.ValueOverrides[1].SecretRef = createTestSecretKeySelector()
	}
	if values != "" {
		vz.Spec.Components.Istio.InstallOverrides.ValueOverrides[2].Values = createTestValueJSON(testIP)
	}
	return vz
}

func createv1alpha1VZOverrides(vz vzapi.Verrazzano, configMap, secret, values, testIP string) vzapi.Verrazzano {
	if configMap != "" {
		vz.Spec.Components.Istio.InstallOverrides.ValueOverrides[0].ConfigMapRef = createTestConfigMapKeySelector()
	}
	if secret != "" {
		vz.Spec.Components.Istio.InstallOverrides.ValueOverrides[1].SecretRef = createTestSecretKeySelector()
	}
	if values != "" {
		vz.Spec.Components.Istio.InstallOverrides.ValueOverrides[2].Values = createTestValueJSON(testIP)
	}
	return vz
}

func createTestConfigMapKeySelector() *corev1.ConfigMapKeySelector {
	return &corev1.ConfigMapKeySelector{
		Key: "configMapKey",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "testCMName",
		},
	}
}

func createTestSecretKeySelector() *corev1.SecretKeySelector {
	return &corev1.SecretKeySelector{
		Key: "secretKey",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "testSecretName",
		},
	}
}

func createTestValueJSON(testIP string) *apiextensionsv1.JSON {
	return &apiextensionsv1.JSON{
		Raw: createValueOverrideData(testIP),
	}
}

func createTestConfigMap(isArrayOfIPs bool, testIPA, testIPB string) *corev1.ConfigMap {
	data := make(map[string]string)
	data["configMapKey"] = createOverrideData(isArrayOfIPs, testIPA, testIPB)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "testCMName",
		},
		Data: data,
	}
}

func createTestSecret(isArrayOfIPs bool, testIPA, testIPB string) *corev1.Secret {
	data := make(map[string][]byte)
	data["secretKey"] = []byte(createOverrideData(isArrayOfIPs, testIPA, testIPB))
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testSecretName",
			Namespace: "default",
		},
		Data: data,
	}
}

// createValueOverrideData returns an Istio override in json format
func createValueOverrideData(externalIP string) []byte {
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

// createOverrideData returns an Istio override with either 1 or 2 externalIPs in string format
func createOverrideData(isArrayOfIPs bool, testIPA, testIPB string) string {
	var data string
	if !isArrayOfIPs {
		data = fmt.Sprintf(`spec:
  components:
    ingressGateways:
    - k8s:
        service:
          externalIPs:
          - ` + testIPA + `
          type: NodePort
      name: istio-ingressgateway`)
	} else {
		data = fmt.Sprintf(`spec:
  components:
    ingressGateways:
    - k8s:
        service:
          externalIPs:
          - ` + testIPA + `
          - ` + testIPB + `
          type: NodePort
      name: istio-ingressgateway`)
	}
	return data
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	return scheme
}

func createFakeTestClient() {
	GetControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects().Build(), nil
	}
}

func createFakeTestClientWithConfigMap(testConfigMap *corev1.ConfigMap) {
	GetControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(testConfigMap).Build(), nil
	}
}

func createFakeTestClientWithSecret(testSecret *corev1.Secret) {
	GetControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(testSecret).Build(), nil
	}
}

func createFakeTestClientWithConfigMapAndSecret(testConfigMap *corev1.ConfigMap, testSecret *corev1.Secret) {
	GetControllerRuntimeClient = func(scheme *runtime.Scheme) (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(testConfigMap, testSecret).Build(), nil
	}
}
