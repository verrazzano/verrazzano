// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

var testScheme = runtime.NewScheme()

const (
	profileDir      = "../../../../manifests/profiles"
	testBomFilePath = "../../testdata/test_bom.json"
)

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = istioclinet.AddToScheme(testScheme)
	_ = istioclisec.AddToScheme(testScheme)
}

func TestAppendOverrides(t *testing.T) {
	falseValue := false
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	tests := []struct {
		name         string
		description  string
		expectedYAML string
		actualCR     vzapi.Verrazzano
		numKeyValues int
		expectedErr  error
	}{
		{
			name:         "DefaultConfig",
			description:  "Test default configuration of AuthProxy with no overrides",
			expectedYAML: "testdata/authProxyValuesNoOverrides.yaml",
			actualCR:     vzapi.Verrazzano{},
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "OverrideReplicas",
			description:  "Test override of replica count",
			expectedYAML: "testdata/authProxyValuesOverrideReplicas.yaml",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							Kubernetes: &vzapi.AuthProxyKubernetesSection{
								CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
									Replicas: 3,
								},
							},
						},
					},
				},
			},
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "DisableAuthProxy",
			description:  "Test overriding AuthProxy to be disabled",
			expectedYAML: "testdata/authProxyValuesOverrideEnabled.yaml",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						AuthProxy: &vzapi.AuthProxyComponent{
							Enabled: &falseValue,
						},
					},
				},
			},
			numKeyValues: 1,
			expectedErr:  nil,
		},
	}
	defer resetWriteFileFunc()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			asserts := assert.New(t)
			t.Log(test.description)

			fakeClient := createFakeClientWithIngress()
			fakeContext := spi.NewFakeContext(fakeClient, &test.actualCR, false, profileDir)

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
				actualValues := authProxyValues{}
				err := yaml.Unmarshal(data, &actualValues)
				asserts.NoError(err)

				// read in the expected results' data from a file and unmarshal it into a values object
				expectedData, err := ioutil.ReadFile(test.expectedYAML)
				asserts.NoError(err, "Error reading expected values yaml file %s", test.expectedYAML)
				expectedValues := authProxyValues{}
				err = yaml.Unmarshal(expectedData, &expectedValues)
				asserts.NoError(err)

				// Compare the actual and expected values objects
				asserts.Equal(expectedValues, actualValues)
				return nil
			}

			var kvs []bom.KeyValue
			kvs, err := AppendOverrides(fakeContext, "", "", "", kvs)
			if test.expectedErr != nil {
				asserts.Error(err)
				asserts.Equal([]bom.KeyValue{}, kvs)
				return
			}
			asserts.NoError(err)
			asserts.Equal(test.numKeyValues, len(kvs))

			// Check Temp file
			asserts.True(kvs[0].IsFile, "Expected generated AuthProxy overrides first in list of helm args")
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

func createFakeClientWithIngress() client.Client {
	fakeClient := fake.NewFakeClientWithScheme(testScheme,
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: vpoconst.NGINXControllerServiceName, Namespace: globalconst.IngressNamespace},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
			},
			Status: corev1.ServiceStatus{
				LoadBalancer: corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{IP: "11.22.33.44"},
					},
				},
			},
		},
	)
	return fakeClient
}

//cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	if err := vzos.RemoveTempFiles(ctx.Log().GetZapLogger(), tmpFileCleanPattern); err != nil {
		ctx.Log().Errorf("Failed deleting temp files: %v", err)
	}
}
