// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const (
	profileDir      = "../../../../../manifests/profiles"
	testBomFilePath = "../../../testdata/test_bom.json"
)

var (
	testScheme = runtime.NewScheme()

	falseValue = false
	trueValue  = true
)

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestIsJaegerOperatorReady tests the isJaegerOperatorReady function for the Jaeger Operator
func TestIsJaegerOperatorReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Jaeger Operator deployment exists and there are available replicas
			// WHEN we call isJaegerOperatorReady
			// THEN the call returns true
			name: "Test IsReady when Jaeger Operator is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
						Labels:    map[string]string{"name": "jaeger-operator"},
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"name": "jaeger-operator"},
						},
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   1,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName + "-95d8c5d96-m6mbr",
						Labels: map[string]string{
							"pod-template-hash": "95d8c5d96",
							"name":              "jaeger-operator",
						},
					},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ComponentNamespace,
						Name:        deploymentName + "-95d8c5d96",
						Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
					},
				},
			).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Jaeger Operator deployment exists and there are no available replicas
			// WHEN we call isJaegerOperatorReady
			// THEN the call returns false
			name: "Test IsReady when Jaeger Operator deployment is not ready",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      deploymentName,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 0,
						Replicas:          1,
						UpdatedReplicas:   0,
					},
				}).Build(),
			expectTrue: false,
		},
		{
			// GIVEN the Jaeger Operator deployment does not exist
			// WHEN we call isJaegerOperatorReady
			// THEN the call returns false
			name:       "Test IsReady when Jaeger Operator deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)
			assert.Equal(t, tt.expectTrue, isJaegerOperatorReady(ctx))
		})
	}
}

// TestPreInstall tests the preInstall function.
func TestPreInstall(t *testing.T) {
	// GIVEN the Jaeger Operator is being installed
	// WHEN the preInstall function is called
	// THEN the component namespace is created in the cluster
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)

	err := preInstall(ctx)
	assert.NoError(t, err)

	ns := corev1.Namespace{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
	assert.NoError(t, err)
}

// TestValidateJaegerOperator tests the validation of the Jaeger Operator installation and the Verrazzano CR
func TestValidateJaegerOperator(t *testing.T) {
	tests := []struct {
		name        string
		vz          vzapi.Verrazzano
		expectError bool
	}{
		{
			name:        "test nothing enabled",
			vz:          vzapi.Verrazzano{},
			expectError: false,
		},
		{
			name: "test jaeger operator enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{Enabled: &trueValue},
					},
				},
			},
			expectError: false,
		},
		{
			name: "test jaeger operator disabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						JaegerOperator: &vzapi.JaegerOperatorComponent{Enabled: &falseValue},
					},
				},
			},
			expectError: false,
		},
	}
	c := jaegerOperatorComponent{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.validateJaegerOperator(&tt.vz)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAppendOverrides tests the AppendOverrides function
// GIVEN a call to AppendOverrides
//  WHEN I call with a ComponentContext with different profiles and overrides
//  THEN the correct overrides file is generated
//
// For each test case a Verrazzano custom resource with different overrides
// is passed into AppendOverrides.  A overrides file is generated by AppendOverrides.
// The test compares the generated and expected overrides files.
func TestAppendOverrides(t *testing.T) {
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
			fakeContext := spi.NewFakeContext(fakeClient, &testCR, false, profileDir)

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
			_, err = os.Stat(tempFilePath)
			asserts.NoError(err, "Unexpected error checking for temp file %s: %s", tempFilePath, err)
			cleanTempFiles(fakeContext)
		})
	}
	// Verify temp files are deleted
	files, err := ioutil.ReadDir(os.TempDir())
	assert.NoError(t, err, "Error reading temp dir to verify file cleanup")
	for _, file := range files {
		assert.False(t,
			strings.HasPrefix(file.Name(), tmpFilePrefix) && strings.HasSuffix(file.Name(), ".yaml"),
			"Found unexpected temp file remaining: %s", file.Name())
	}

}

//cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	if err := vzos.RemoveTempFiles(ctx.Log().GetZapLogger(), tmpFileCleanPattern); err != nil {
		ctx.Log().Errorf("Failed deleting temp files: %v", err)
	}
}

// TestEnsureMonitoringOperatorNamespace asserts the verrazzano-monitoring namespaces can be created
func TestEnsureMonitoringOperatorNamespace(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), jaegerEnabledCR, false)
	err := ensureVerrazzanoMonitoringNamespace(ctx)
	assert.NoError(t, err)
}
