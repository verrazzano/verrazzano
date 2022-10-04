// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"

	"io/fs"
	"io/ioutil"
	"os"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const (
	profileDir         = "../../../../../manifests/profiles"
	testBomFilePath    = "../../../testdata/test_bom.json"
	jaegerDisabledJSON = "{\"jaeger\": {\"create\": false}}"
)

var (
	testScheme = runtime.NewScheme()

	falseValue = false
	trueValue  = true
)

var jaegerDisabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			JaegerOperator: &vzapi.JaegerOperatorComponent{
				Enabled: &trueValue,
				InstallOverrides: vzapi.InstallOverrides{
					MonitorChanges: &trueValue,
					ValueOverrides: []vzapi.Overrides{
						{
							Values: &apiextensionsv1.JSON{
								Raw: []byte(jaegerDisabledJSON),
							},
						},
					},
				},
			},
		},
	},
}

var keycloakDisabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Keycloak: &vzapi.KeycloakComponent{
				Enabled: &falseValue,
			},
		},
	},
}

var keycloakEnabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Keycloak: &vzapi.KeycloakComponent{
				Enabled: &trueValue,
			},
		},
	},
}

var vzEsInternalSecret = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      globalconst.VerrazzanoESInternal,
		Namespace: constants.VerrazzanoSystemNamespace,
	},
}

var vzEsInternalSecretWithData = &corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      globalconst.VerrazzanoESInternal,
		Namespace: constants.VerrazzanoSystemNamespace,
	},
	Data: map[string][]byte{
		"username": []byte("verrazzano"),
		"password": []byte("dummy"),
	},
}

var vzIngressService = &corev1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Name:      constants.NGINXControllerServiceName,
		Namespace: constants.IngressNginxNamespace,
	},
	Spec: corev1.ServiceSpec{
		ExternalIPs: []string{"127.0.0.1"},
	},
}

type preInstallTestStruct struct {
	name   string
	spec   *vzapi.Verrazzano
	client client.Client
	err    error
	dryRun bool
}

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestPreInstall tests the preInstall function for various scenarios.
func TestPreInstallInternal(t *testing.T) {
	for _, tt := range getPreInstallTests() {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, tt.spec, nil, tt.dryRun)
			err := preInstall(ctx)
			if tt.err != nil {
				assert.Error(t, err)
				assert.IsTypef(t, tt.err, err, "")
			} else {
				assert.NoError(t, err)
			}
			ns := corev1.Namespace{}
			if !tt.dryRun {
				err = tt.client.Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
				assert.NoError(t, err)
			}
		})
	}
}

// TestAppendOverrides tests the AppendOverrides function
// GIVEN a call to AppendOverrides
//
//	WHEN I call with a ComponentContext with different profiles and overrides
//	THEN the correct overrides file is generated
//
// For each test case a Verrazzano custom resource with different overrides
// is passed into AppendOverrides.  A overrides file is generated by AppendOverrides.
// The test compares the generated and expected overrides files.
func TestAppendOverrides(t *testing.T) {
	var files []string
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	tests := []struct {
		name         string
		description  string
		expectedYAML string
		actualCR     string
		numKeyValues int
		expectedErr  error
	}{
		{
			name:         "OverrideJaegerImages",
			description:  "Test overriding Jaeger Images",
			expectedYAML: "testdata/jaegerOperatorOverrideValues.yaml",
			actualCR:     "testdata/jaegerOperatorOverrideVz.yaml",
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "OverrideJaegerCreate",
			description:  "Test overriding Jaeger create",
			expectedYAML: "testdata/jaegerOperatorJaegerCreateDisabledValues.yaml",
			actualCR:     "testdata/jaegerOperatorOverrideJaegerCreateVz.yaml",
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "OverrideMetricsStorageType",
			description:  "Test overriding metrics storage type",
			expectedYAML: "testdata/jaegerOperatorOverrideValues.yaml",
			actualCR:     "testdata/jaegerOperatorOverrideMetricsStorageVz.yaml",
			numKeyValues: 2,
			expectedErr:  nil,
		},
		{
			name:         "OpenSearchDisabled",
			description:  "Test OpenSearch disabled",
			expectedYAML: "testdata/jaegerOperatorJaegerCreateDisabledValues.yaml",
			actualCR:     "testdata/jaegerOperatorDisableOpenSearchVz.yaml",
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "OpenSearchReplicasInProdProfile",
			description:  "Test OpenSearch replica count setting in prod profile",
			expectedYAML: "testdata/jaegerOperatorProdOverrideValues.yaml",
			actualCR:     "testdata/jaegerOperatorProdOverrideVz.yaml",
			numKeyValues: 1,
			expectedErr:  nil,
		},
	}
	defer resetWriteFileFunc()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			asserts := assert.New(t)
			t.Log(test.description)

			// Read the Verrazzano CR into a struct
			testCR := vzapi.Verrazzano{}
			yamlFile, err := ioutil.ReadFile(test.actualCR)
			asserts.NoError(err)
			err = yaml.Unmarshal(yamlFile, &testCR)
			asserts.NoError(err)

			fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
			fakeContext := spi.NewFakeContext(fakeClient, &testCR, nil, false, profileDir)

			writeFileFunc = func(filename string, data []byte, perm fs.FileMode) error {
				if test.expectedErr != nil {
					return test.expectedErr
				}
				if err := ioutil.WriteFile(filename, data, perm); err != nil {
					asserts.Failf("Failure writing file %s: %s", filename, err)
					return err
				}
				asserts.FileExists(filename)

				// Unmarshal the actual generated helm values from code under test
				actualJSON, err := yaml.YAMLToJSON(data)
				asserts.NoError(err)

				// read in the expected results' data from a file and unmarshal it into a values object
				expectedData, err := ioutil.ReadFile(test.expectedYAML)
				asserts.NoError(err, "Error reading expected values yaml file %s", test.expectedYAML)
				expectedJSON, err := yaml.YAMLToJSON(expectedData)
				asserts.NoError(err)

				// Compare the actual and expected values objects
				asserts.Equal(string(expectedJSON), string(actualJSON))
				return nil
			}

			var kvs []bom.KeyValue
			kvs, err = AppendOverrides(fakeContext, "", "", "", kvs)
			if test.expectedErr != nil {
				asserts.Error(err)
				asserts.Equal([]bom.KeyValue{}, kvs)
				return
			}
			asserts.NoError(err)
			asserts.Equal(test.numKeyValues, len(kvs))

			// Check Temp file
			asserts.True(kvs[0].IsFile, "Expected generated Jaeger Operator overrides first in list of helm args")
			tempFilePath := kvs[0].Value
			files = append(files, tempFilePath)
			_, err = os.Stat(tempFilePath)
			asserts.NoError(err, "Unexpected error checking for temp file %s: %v", tempFilePath, err)
			err = cleanFile(tempFilePath)
			asserts.NoError(err, "Unexpected error when cleaning up temp files %s: %v", tempFilePath, err)

			if test.name == "OverrideMetricsStorageType" {
				asserts.Equal(kvs[1].Key, prometheusServerField)
				asserts.Equal(kvs[1].Value, prometheusURL)
			}
		})
	}
	// Verify temp files are deleted
	for _, file := range files {
		assert.False(t, fileExists(file),
			"Found unexpected temp file remaining: %s", file)
	}

}

// cleanFile - Clean up the specified file path
func cleanFile(file string) error {
	if err := os.Remove(file); err != nil {
		return err
	}
	return nil
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}

// TestBuildJaegerDNSNames asserts if the generated DNS name for Jaeger is correct.
func TestBuildJaegerDNSNames(t *testing.T) {
	// GIVEN a Verrazzano CR with Jaeger Component enabled,
	// WHEN we call the buildJaegerHostnameForDomain function,
	// THEN correct FQDN for Jaeger is returned.
	jaegerDNSName := buildJaegerHostnameForDomain("default.nip.io")
	assert.Equal(t, "jaeger.default.nip.io", jaegerDNSName)
}

func createFakeClient(extraObjs ...client.Object) client.Client {
	var objs []client.Object
	objs = append(objs, extraObjs...)
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(objs...).Build()
	return c
}

func getPreInstallTests() []preInstallTestStruct {
	return []preInstallTestStruct{
		// GIVEN a Verrazzano CR with Keycloak Component enabled,
		// WHEN we call the PreInstall function with no secret to access the storage,
		// THEN an error is returned.
		{
			"should fail when verrazzano-es-internal secret does not exist and keycloak is enabled",
			keycloakEnabledCR,
			createFakeClient(),
			ctrlerrors.RetryableError{Source: ComponentName},
			false,
		},
		// GIVEN a Verrazzano CR with Keycloak Component enabled,
		// WHEN we call the PreInstall function with a valid secret to access the storage,
		// THEN no error is returned.
		{
			"should pass when verrazzano-es-internal secret does exist without data and keycloak is enabled",
			keycloakEnabledCR,
			createFakeClient(vzEsInternalSecret),
			nil,
			false,
		},
		// GIVEN a Verrazzano CR with Keycloak Component enabled,
		// WHEN we call the PreInstall function with a valid secret to access the storage,
		// THEN no error is returned.
		{
			"should pass when verrazzano-es-internal secret does exist with valid data and keycloak is enabled",
			keycloakEnabledCR,
			createFakeClient(vzEsInternalSecretWithData),
			nil,
			false,
		},
		// GIVEN a Verrazzano CR with Keycloak Component enabled,
		// WHEN we call the PreInstall function with/without a valid secret to access the storage,
		// THEN no error is returned.
		{
			"always nil error when keycloak is disabled",
			keycloakDisabledCR,
			createFakeClient(),
			nil,
			false,
		},
		// GIVEN a Verrazzano CR with Jaeger Component disabled and dry run is false,
		// WHEN we call the PreInstall function with a valid secret to access the storage,
		// THEN no error is returned.
		{
			"always nil error when jaeger instance creation is disabled",
			jaegerDisabledCR,
			createFakeClient(),
			nil,
			false,
		},
		// GIVEN a Verrazzano CR with Jaeger Component disabled and dry run is true,
		// WHEN we call the PreInstall function with a valid secret to access the storage,
		// THEN no error is returned.
		{
			"always nil error when it is a dry run",
			jaegerDisabledCR,
			createFakeClient(),
			nil,
			true,
		},
	}
}

func TestGetOverrides(t *testing.T) {
	ref := &corev1.ConfigMapKeySelector{
		Key: "foo",
	}
	o := v1beta1.InstallOverrides{
		ValueOverrides: []v1beta1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	oV1Alpha1 := vzapi.InstallOverrides{
		ValueOverrides: []vzapi.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	var tests = []struct {
		name string
		cr   runtime.Object
		res  interface{}
	}{
		{
			"overrides when component not nil, v1alpha1",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{
							InstallOverrides: oV1Alpha1,
						},
					},
				},
			},
			oV1Alpha1.ValueOverrides,
		},
		{
			"Empty overrides when component nil",
			&v1beta1.Verrazzano{},
			[]v1beta1.Overrides{},
		},
		{
			"overrides when component not nil",
			&v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						JaegerOperator: &v1beta1.JaegerOperatorComponent{
							InstallOverrides: o,
						},
					},
				},
			},
			o.ValueOverrides,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			override := GetOverrides(tt.cr)
			assert.EqualValues(t, tt.res, override)
		})
	}
}
