// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const (
	profileDir      = "../../../../manifests/profiles"
	testBomFilePath = "../../testdata/test_bom.json"
)

var (
	testScheme  = runtime.NewScheme()
	pvc100Gi, _ = resource.ParseQuantity("100Gi")
)

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vmov1.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = vzclusters.AddToScheme(testScheme)

	_ = istioclinet.AddToScheme(testScheme)
	_ = istioclisec.AddToScheme(testScheme)
	_ = certv1.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// TestVzResolveNamespace tests the Verrazzano component name
// GIVEN a Verrazzano component
//  WHEN I call resolveNamespace
//  THEN the Verrazzano namespace name is correctly resolved
func TestVzResolveNamespace(t *testing.T) {
	const defNs = vpoconst.VerrazzanoSystemNamespace
	a := assert.New(t)
	ns := resolveVerrazzanoNamespace("")
	a.Equal(defNs, ns, "Wrong namespace resolved for Verrazzano when using empty namespace")
	ns = resolveVerrazzanoNamespace("default")
	a.Equal(defNs, ns, "Wrong namespace resolved for Verrazzano when using default namespace")
	ns = resolveVerrazzanoNamespace("custom")
	a.Equal("custom", ns, "Wrong namespace resolved for Verrazzano when using custom namesapce")
}

// Test_appendVerrazzanoValues tests the appendVerrazzanoValues function
// GIVEN a call to appendVerrazzanoValues
//  WHEN I call with a ComponentContext with different profiles and overrides
//  THEN the correct KeyValue objects and overrides file snippets are generated
func Test_appendVerrazzanoValues(t *testing.T) {
	falseValue := false
	trueValue := true
	tests := []struct {
		name         string
		description  string
		expectedYAML string
		actualCR     vzapi.Verrazzano
		expectedErr  error
	}{
		{
			name:         "BasicProdVerrazzanoNoOverrides",
			description:  "Test basic prod no user overrides",
			actualCR:     vzapi.Verrazzano{},
			expectedYAML: "testdata/vzValuesProdNoOverrides.yaml",
			expectedErr:  nil,
		},
		{
			name:         "BasicDevVerrazzanoNoOverrides",
			description:  "Test basic dev no user overrides",
			actualCR:     vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Profile: "dev"}},
			expectedYAML: "testdata/vzValuesDevNoOverrides.yaml",
			expectedErr:  nil,
		},
		{
			name:         "BasicManagedClusterVerrazzanoNoOverrides",
			description:  "Test basic managed-cluster no user overrides",
			actualCR:     vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Profile: "managed-cluster"}},
			expectedYAML: "testdata/vzValuesMgdClusterNoOverrides.yaml",
			expectedErr:  nil,
		},
		{
			name:        "DevVerrazzanoWithOverrides",
			description: "Test dev profile with overrides no user overrides",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile:         "dev",
					EnvironmentName: "myenv",
					Components: vzapi.ComponentSpec{
						Prometheus:             &vzapi.PrometheusComponent{Enabled: &falseValue},
						Kibana:                 &vzapi.KibanaComponent{Enabled: &falseValue},
						Elasticsearch:          &vzapi.ElasticsearchComponent{Enabled: &falseValue},
						Grafana:                &vzapi.GrafanaComponent{Enabled: &falseValue},
						Keycloak:               &vzapi.KeycloakComponent{Enabled: &falseValue},
						Rancher:                &vzapi.RancherComponent{Enabled: &falseValue},
						DNS:                    &vzapi.DNSComponent{Wildcard: &vzapi.Wildcard{Domain: "xip.io"}},
						PrometheusOperator:     &vzapi.PrometheusOperatorComponent{Enabled: &trueValue},
						PrometheusAdapter:      &vzapi.PrometheusAdapterComponent{Enabled: &trueValue},
						KubeStateMetrics:       &vzapi.KubeStateMetricsComponent{Enabled: &trueValue},
						PrometheusPushgateway:  &vzapi.PrometheusPushgatewayComponent{Enabled: &trueValue},
						PrometheusNodeExporter: &vzapi.PrometheusNodeExporterComponent{Enabled: &trueValue},
						JaegerOperator:         &vzapi.JaegerOperatorComponent{Enabled: &trueValue},
					},
				},
			},
			expectedYAML: "testdata/vzValuesDevWithOverrides.yaml",
			expectedErr:  nil,
		},
		{
			name:        "ProdWithExternaDNSEnabled",
			description: "Test prod with OCI DNS enabled, should enable external-dns component",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret:        "myOCISecret",
								DNSZoneCompartmentOCID: "myCompartmentOCID",
								DNSZoneOCID:            "myZoneOCID",
								DNSZoneName:            "myzone.com",
							},
						},
					},
				},
			},
			expectedYAML: "testdata/vzValuesProdWithExternalDNS.yaml",
			expectedErr:  nil,
		},
	}
	defer resetWriteFileFunc()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)

			t.Log(test.description)

			fakeClient := createFakeClientWithIngress()
			fakeContext := spi.NewFakeContext(fakeClient, &test.actualCR, nil, false, profileDir)
			values := verrazzanoValues{}

			writeFileFunc = func(filename string, data []byte, perm fs.FileMode) error {
				if test.expectedErr != nil {
					return test.expectedErr
				}
				a.Equal([]byte(test.expectedYAML), data)
				return nil
			}

			err := appendVerrazzanoValues(fakeContext, &values)
			if test.expectedErr != nil {
				a.Error(err)
			} else {
				a.NoError(err)
			}

			// outdata, err := yaml.Marshal(&values)
			// assert.NoError(err)
			// ioutil.WriteFile(fmt.Sprintf("%s/%s.yaml", os.TempDir(), test.name), outdata, fs.FileMode(0664))

			data, err := ioutil.ReadFile(test.expectedYAML)
			a.NoError(err, "Error reading expected values yaml file %s", test.expectedYAML)
			expectedValues := verrazzanoValues{}
			err = yaml.Unmarshal(data, &expectedValues)
			a.NoError(err)
			a.Equal(expectedValues, values)
		})
	}
}

// Test_appendVMIValues tests the appendVMIValues function
// GIVEN a call to appendVMIValues
//  WHEN I call with a ComponentContext with different profiles and overrides
//  THEN the correct KeyValue objects and overrides file snippets are generated
func Test_appendVMIValues(t *testing.T) {
	falseValue := false
	tests := []struct {
		name                  string
		description           string
		expectedYAML          string
		actualCR              vzapi.Verrazzano
		expectedHelmOverrides []bom.KeyValue
		expectedErr           error
	}{
		{
			name:                  "VMIProdVerrazzanoNoOverrides",
			description:           "Test VMI basic prod no user overrides",
			actualCR:              vzapi.Verrazzano{},
			expectedYAML:          "testdata/vzValuesVMIProdVerrazzanoNoOverrides.yaml",
			expectedHelmOverrides: []bom.KeyValue{},
			expectedErr:           nil,
		},
		{
			name:                  "VMIDevVerrazzanoNoOverrides",
			description:           "Test VMI basic dev no user overrides",
			actualCR:              vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Profile: "dev"}},
			expectedYAML:          "testdata/vzValuesVMIDevVerrazzanoNoOverrides.yaml",
			expectedHelmOverrides: []bom.KeyValue{},
			expectedErr:           nil,
		},
		{
			name:                  "VMIManagedClusterVerrazzanoNoOverrides",
			description:           "Test VMI basic managed-cluster no user overrides",
			actualCR:              vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Profile: "managed-cluster"}},
			expectedYAML:          "testdata/vzValuesVMIManagedClusterVerrazzanoNoOverrides.yaml",
			expectedHelmOverrides: []bom.KeyValue{},
			expectedErr:           nil,
		},
		{
			name:        "VMIDevWithOverrides",
			description: "Test VMI dev profile with overrides no user overrides",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile: "dev",
					Components: vzapi.ComponentSpec{
						Grafana:       &vzapi.GrafanaComponent{Enabled: &falseValue},
						Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &falseValue},
						Prometheus:    &vzapi.PrometheusComponent{Enabled: &falseValue},
						Kibana:        &vzapi.KibanaComponent{Enabled: &falseValue},
					},
				},
			},
			expectedYAML:          "testdata/vzValuesVMIDevWithOverrides.yaml",
			expectedHelmOverrides: []bom.KeyValue{},
			expectedErr:           nil,
		},
		{
			name:        "VMIDevWithStorageOverrides",
			description: "Test VMI dev profile with overrides no user overrides",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile:             "dev",
					DefaultVolumeSource: &corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "vmi"}},
					VolumeClaimSpecTemplates: []vzapi.VolumeClaimSpecTemplate{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "vmi"},
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										"storage": pvc100Gi,
									},
								},
							},
						},
					},
					Components: vzapi.ComponentSpec{},
				},
			},
			expectedYAML:          "testdata/vzValuesVMIDevWithStorageOverrides.yaml",
			expectedHelmOverrides: []bom.KeyValue{},
			expectedErr:           nil,
		},
		{
			name:        "VMIProdWithStorageOverrides",
			description: "Test VMI prod profile with emptyDir defaultVolumeSource override",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile:             "prod",
					DefaultVolumeSource: &corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				},
			},
			expectedYAML:          "testdata/vzValuesVMIProdWithStorageOverrides.yaml",
			expectedHelmOverrides: []bom.KeyValue{},
			expectedErr:           nil,
		},
		{
			name:        "VMIProdWithESInstallArgs",
			description: "Test VMI prod profile with emptyDir defaultVolumeSource override",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile:             "prod",
					DefaultVolumeSource: &corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							ESInstallArgs: []vzapi.InstallArgs{
								{Name: "nodes.master.replicas", Value: "6"},
								{Name: "nodes.master.requests.memory", Value: "3G"},
								{Name: "nodes.ingest.replicas", Value: "8"},
								{Name: "nodes.ingest.requests.memory", Value: "32G"},
								{Name: "nodes.data.replicas", Value: "16"},
								{Name: "nodes.data.requests.memory", Value: "32G"},
							},
						},
					},
				},
			},
			expectedHelmOverrides: []bom.KeyValue{},
			expectedYAML:          "testdata/vzValuesVMIProdWithESInstallArgs.yaml",
			expectedErr:           nil,
		},
	}
	defer resetWriteFileFunc()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)

			t.Log(test.description)

			fakeClient := createFakeClientWithIngress()
			fakeContext := spi.NewFakeContext(fakeClient, &test.actualCR, nil, false, profileDir)
			values := verrazzanoValues{}

			writeFileFunc = func(filename string, data []byte, perm fs.FileMode) error {
				if test.expectedErr != nil {
					return test.expectedErr
				}
				a.Equal([]byte(test.expectedYAML), data)
				return nil
			}

			storageOverride, err := common.FindStorageOverride(fakeContext.EffectiveCR())
			a.NoError(err)

			keyValues, err := appendVMIOverrides(fakeContext.EffectiveCR(), &values, storageOverride, []bom.KeyValue{})
			a.NoError(err)
			a.Equal(test.expectedHelmOverrides, keyValues, "Install args did not match")

			data, err := ioutil.ReadFile(test.expectedYAML)
			a.NoError(err, "Error reading expected values yaml file %s", test.expectedYAML)
			expectedValues := verrazzanoValues{}
			err = yaml.Unmarshal(data, &expectedValues)
			a.NoError(err)
			a.Equal(expectedValues, values)
		})
	}
}

// Test_appendVerrazzanoOverrides tests the appendVerrazzanoOverrides function
// GIVEN a call to appendVerrazzanoOverrides
//  WHEN I call with a ComponentContext with different profiles and overrides
//  THEN the correct KeyValue objects and overrides file snippets are generated
func Test_appendVerrazzanoOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() {
		config.SetDefaultBomFilePath("")
	}()
	falseValue := false
	trueValue := true
	tests := []struct {
		name         string
		description  string
		expectedYAML string
		actualCR     vzapi.Verrazzano
		expectedErr  error
		numKeyValues int
	}{
		{
			name:         "ProdDefault",
			description:  "Test basic prod profile with no user overrides",
			actualCR:     vzapi.Verrazzano{},
			expectedYAML: "testdata/vzOverridesProdDefault.yaml",
		},
		{
			name:        "ProdDefaultIOError",
			description: "Test basic prod profile with no user overrides",
			actualCR:    vzapi.Verrazzano{},
			expectedErr: fmt.Errorf("Error writing file"),
		},
		{
			name:         "DevDefault",
			description:  "Test basic dev profile with no user overrides",
			actualCR:     vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Profile: "dev"}},
			expectedYAML: "testdata/vzOverridesDevDefault.yaml",
			numKeyValues: 1,
		},
		{
			name:         "ManagedClusterDefault",
			description:  "Test basic managed-cluster no user overrides",
			actualCR:     vzapi.Verrazzano{Spec: vzapi.VerrazzanoSpec{Profile: "managed-cluster"}},
			expectedYAML: "testdata/vzOverridesManagedClusterDefault.yaml",
			numKeyValues: 1,
		},
		{
			name:        "DevWithOverrides",
			description: "Test dev profile with user overrides",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Profile:         "dev",
					EnvironmentName: "myenv",
					Components: vzapi.ComponentSpec{
						Prometheus:             &vzapi.PrometheusComponent{Enabled: &falseValue},
						Kibana:                 &vzapi.KibanaComponent{Enabled: &falseValue},
						Elasticsearch:          &vzapi.ElasticsearchComponent{Enabled: &falseValue},
						Grafana:                &vzapi.GrafanaComponent{Enabled: &falseValue},
						Keycloak:               &vzapi.KeycloakComponent{Enabled: &falseValue},
						Rancher:                &vzapi.RancherComponent{Enabled: &falseValue},
						DNS:                    &vzapi.DNSComponent{Wildcard: &vzapi.Wildcard{Domain: "xip.io"}},
						PrometheusOperator:     &vzapi.PrometheusOperatorComponent{Enabled: &trueValue},
						PrometheusAdapter:      &vzapi.PrometheusAdapterComponent{Enabled: &trueValue},
						KubeStateMetrics:       &vzapi.KubeStateMetricsComponent{Enabled: &trueValue},
						PrometheusPushgateway:  &vzapi.PrometheusPushgatewayComponent{Enabled: &trueValue},
						PrometheusNodeExporter: &vzapi.PrometheusNodeExporterComponent{Enabled: &trueValue},
						JaegerOperator:         &vzapi.JaegerOperatorComponent{Enabled: &trueValue},
					},
				},
			},
			expectedYAML: "testdata/vzOverridesDevWithOverrides.yaml",
			numKeyValues: 1,
		},
		{
			name:        "ProdWithExternaDNSEnabled",
			description: "Test prod with OCI DNS enabled, should enable external-dns component",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{
								OCIConfigSecret:        "myOCISecret",
								DNSZoneCompartmentOCID: "myCompartmentOCID",
								DNSZoneOCID:            "myZoneOCID",
								DNSZoneName:            "myzone.com",
							},
						},
					},
				},
			},
			expectedYAML: "testdata/vzOverridesProdWithExternaDNSEnabled.yaml",
		},
		{
			name:        "ProdWithAdminRoleOverrides",
			description: "Test prod with Security admin role overrides only",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Security: vzapi.SecuritySpec{
						AdminSubjects: []rbacv1.Subject{
							{
								Kind: "User",
								Name: "kilgore-trout",
							},
							{
								Kind: "User",
								Name: "fred-flintstone",
							},
						},
					},
				},
			},
			expectedYAML: "testdata/vzOverridesProdWithAdminRoleOverrides.yaml",
		},
		{
			name:        "ProdWithMonitorRoleOverrides",
			description: "Test prod with Monitor admin role overrides only",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Security: vzapi.SecuritySpec{
						MonitorSubjects: []rbacv1.Subject{
							{
								Kind: "Group",
								Name: "group-of-monitors",
							},
							{
								Kind: "User",
								Name: "joe-monitor",
							},
						},
					},
				},
			},
			expectedYAML: "testdata/vzOverridesProdWithMonitorRoleOverrides.yaml",
		},
		{
			name:        "ProdWithAdminAndMonitorRoleOverrides",
			description: "Test prod with Security admin and monitor role overrides",
			actualCR: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Security: vzapi.SecuritySpec{
						AdminSubjects: []rbacv1.Subject{
							{
								Kind: "User",
								Name: "kilgore-trout",
							},
						},
						MonitorSubjects: []rbacv1.Subject{
							{
								Kind: "Group",
								Name: "group-of-monitors",
							},
						},
					},
				},
			},
			expectedYAML: "testdata/vzOverridesProdWithAdminAndMonitorRoleOverrides.yaml",
		},
	}
	defer resetWriteFileFunc()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			t.Log(test.description)

			fakeClient := createFakeClientWithIngress()
			fakeContext := spi.NewFakeContext(fakeClient, &test.actualCR, nil, false, profileDir)

			writeFileFunc = func(filename string, data []byte, perm fs.FileMode) error {
				if test.expectedErr != nil {
					return test.expectedErr
				}
				if err := ioutil.WriteFile(filename, data, perm); err != nil {
					a.Failf("Failure writing file %s: %s", filename, err)
					return err
				}
				a.FileExists(filename)

				// Unmarshal the VZ expected and actual data into verrazzanoValues structs
				// and do a deep-equals comparison using the asserts package

				// Unmarshal the actual generated helm values from code under test
				actualValues := verrazzanoValues{}
				err := yaml.Unmarshal(data, &actualValues)
				a.NoError(err)

				// read in the expected results data from a file and unmarshal it into a values object
				expectedData, err := ioutil.ReadFile(test.expectedYAML)
				a.NoError(err, "Error reading expected values yaml file %s", test.expectedYAML)
				expectedValues := verrazzanoValues{}
				err = yaml.Unmarshal(expectedData, &expectedValues)

				a.NoError(err)
				// Compare the actual and expected values objects
				a.Equal(expectedValues, actualValues)
				a.Equal(HashSum(expectedValues), HashSum(actualValues))
				return nil
			}

			kvs := []bom.KeyValue{}
			kvs, err := appendVerrazzanoOverrides(fakeContext, "", "", "", kvs)
			if test.expectedErr != nil {
				a.Error(err)
				a.Equal([]bom.KeyValue{}, kvs)
				return
			}
			a.NoError(err)

			actualNumKvs := len(kvs)
			expectedNumKvs := test.numKeyValues
			if expectedNumKvs == 0 {
				// default is 1 custom image overrides
				expectedNumKvs = 1
			}
			a.Equal(expectedNumKvs, actualNumKvs)
			// Check Temp file
			a.True(kvs[0].IsFile, "Expected generated verrazzano overrides first in list of helm args")
			tempFilePath := kvs[0].Value
			_, err = os.Stat(tempFilePath)
			a.NoError(err, "Unexpected error checking for temp file %s: %s", tempFilePath, err)
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

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
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
	).Build()
	return fakeClient
}

// TestFakeExecHandler is a test intended to be use to handle fake command execution
// See the fakeExecCommand function.
// When this test is invoked normally no TEST_FAKE_EXEC_SCENARIO is present
// so no assertions are made and therefore passes.
func TestFakeExecHandler(t *testing.T) {
	a := assert.New(t)
	scenario, found := os.LookupEnv("TEST_FAKE_EXEC_SCENARIO")
	if found {
		switch scenario {
		case "fixupElasticSearchReplicaCount/get":
			a.Equal(`curl -v -XGET -s -k --fail http://localhost:42/_cluster/health`,
				os.Args[13], "Expected curl command to be correct.")
			fmt.Print(`"number_of_data_nodes":1,`)
		case "fixupElasticSearchReplicaCount/put":
			fmt.Println(scenario)
			fmt.Println(strings.Join(os.Args, " "))
			a.Equal(`curl -v -XPUT -d '{"index":{"auto_expand_replicas":"0-1"}}' --header 'Content-Type: application/json' -s -k --fail http://localhost:42/verrazzano-*/_settings`,
				os.Args[13], "Expected curl command to be correct.")
		default:
			a.Fail("Unknown test scenario provided in environment variable TEST_FAKE_EXEC_SCENARIO: %s", scenario)
		}
	}
}

// TestAssociateHelmObjectToThisRelease tests labelling/annotating objects that will be imported to a helm chart
// GIVEN an unmanaged object
//  WHEN I call associateHelmObjectToThisRelease
//  THEN the object is managed by helm
func TestAssociateHelmObjectToThisRelease(t *testing.T) {
	namespacedName := types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	obj := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}

	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(obj).Build()
	_, err := associateHelmObjectToThisRelease(c, obj, namespacedName)
	assert.NoError(t, err)
	assert.Equal(t, obj.Annotations["meta.helm.sh/release-name"], ComponentName)
	assert.Equal(t, obj.Annotations["meta.helm.sh/release-namespace"], globalconst.VerrazzanoSystemNamespace)
	assert.Equal(t, obj.Labels["app.kubernetes.io/managed-by"], "Helm")
}

// TestAssociateHelmObjectAndKeep tests labelling/annotating objects that will be associated to a helm chart
// GIVEN an unmanaged object
//  WHEN I call associateHelmObject with keep set to true
//  THEN the object is managed by helm and is labeled with a resource policy of "keep"
func TestAssociateHelmObjectAndKeep(t *testing.T) {
	namespacedName := types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	obj := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}

	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(obj).Build()
	_, err := common.AssociateHelmObject(c, obj, namespacedName, namespacedName, true)
	assert.NoError(t, err)
	assert.Equal(t, ComponentName, obj.Annotations["meta.helm.sh/release-name"])
	assert.Equal(t, globalconst.VerrazzanoSystemNamespace, obj.Annotations["meta.helm.sh/release-namespace"])
	assert.Equal(t, "keep", obj.Annotations["helm.sh/resource-policy"])
	assert.Equal(t, "Helm", obj.Labels["app.kubernetes.io/managed-by"])
}

// TestIsReadyNotReady tests the Verrazzano isVerrazzanoReady call
// GIVEN a Verrazzano component
//  WHEN I call isVerrazzanoReady when it is installed but the secret is not found
//  THEN false is returned
func TestIsReadyNotReady(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	assert.False(t, isVerrazzanoReady(ctx))
}

// TestIsReady tests the Verrazzano isVerrazzanoReady call
// GIVEN Verrazzano components that are all enabled by default
//  WHEN I call isVerrazzanoReady when all requirements are met
//  THEN false is returned
func TestIsReady(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
			Namespace: ComponentNamespace}},
	).Build()

	vz := &vzapi.Verrazzano{}
	vz.Spec.Components = vzapi.ComponentSpec{}
	ctx := spi.NewFakeContext(c, vz, nil, false)
	assert.True(t, isVerrazzanoReady(ctx))
}

// TestIsReadyDeploymentVMIDisabled tests the Verrazzano isVerrazzanoReady call
// GIVEN a Verrazzano component with all VMI components disabled
//  WHEN I call isVerrazzanoReady
//  THEN true is returned
func TestIsReadyDeploymentVMIDisabled(t *testing.T) {
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "verrazzano",
		Namespace: ComponentNamespace}},
	).Build()
	vz := &vzapi.Verrazzano{}
	falseValue := false
	vz.Spec.Components = vzapi.ComponentSpec{
		Fluentd:       &vzapi.FluentdComponent{Enabled: &falseValue},
		Kibana:        &vzapi.KibanaComponent{Enabled: &falseValue},
		Elasticsearch: &vzapi.ElasticsearchComponent{Enabled: &falseValue},
		Prometheus:    &vzapi.PrometheusComponent{Enabled: &falseValue},
		Grafana:       &vzapi.GrafanaComponent{Enabled: &falseValue},
	}
	ctx := spi.NewFakeContext(c, vz, nil, false)
	assert.True(t, isVerrazzanoReady(ctx))
}

func TestConfigHashSum(t *testing.T) {
	defaultAppLogID := "test-defaultAppLogId"
	systemLogID := "test-systemLogId"
	apiSec := "test-my-apiSec"
	b := true
	f1 := vzapi.FluentdComponent{
		OCI: &vzapi.OciLoggingConfiguration{DefaultAppLogID: defaultAppLogID,
			SystemLogID: systemLogID, APISecret: apiSec,
		}}
	f2 := vzapi.FluentdComponent{OCI: &vzapi.OciLoggingConfiguration{
		APISecret:       apiSec,
		DefaultAppLogID: defaultAppLogID,
		SystemLogID:     systemLogID,
	}}
	assert.Equal(t, HashSum(f1), HashSum(f2))
	f1.Enabled = &b
	assert.NotEqual(t, HashSum(f1), HashSum(f2))
	f2.Enabled = &b
	assert.Equal(t, HashSum(f1), HashSum(f2))
}

// TestRemoveNodeExporterResources tests the removeNodeExporterResources function
// GIVEN a Verrazzano component
// WHEN I call the removeNodeExporterResources function
// THEN the function removes all of the expected resources from the cluster
func TestRemoveNodeExporterResources(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: monitoringNamespace,
				Name:      nodeExporter,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: monitoringNamespace,
				Name:      nodeExporter,
			},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: monitoringNamespace,
				Name:      nodeExporter,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeExporter,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeExporter,
			},
		},
	).Build()

	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	removeNodeExporterResources(ctx)

	namespacedName := types.NamespacedName{Namespace: monitoringNamespace, Name: nodeExporter}
	s := &corev1.Service{}
	err := c.Get(context.TODO(), namespacedName, s)
	assert.True(t, errors.IsNotFound(err))

	sa := &corev1.ServiceAccount{}
	err = c.Get(context.TODO(), namespacedName, sa)
	assert.True(t, errors.IsNotFound(err))

	ds := &appsv1.DaemonSet{}
	err = c.Get(context.TODO(), namespacedName, ds)
	assert.True(t, errors.IsNotFound(err))

	crb := &rbacv1.ClusterRoleBinding{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: nodeExporter}, crb)
	assert.True(t, errors.IsNotFound(err))

	cr := &rbacv1.ClusterRole{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: nodeExporter}, cr)
	assert.True(t, errors.IsNotFound(err))
}
