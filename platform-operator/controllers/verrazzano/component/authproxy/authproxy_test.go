// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"io/fs"
	"os"
	"strings"
	"testing"

	netv1 "k8s.io/api/networking/v1"

	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

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
	appsv1 "k8s.io/api/apps/v1"
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

// TestIsAuthProxyReady tests the isAuthProxyReady call
// GIVEN a AuthProxy component
//
//	WHEN I call isAuthProxyReady when all requirements are met
//	THEN true or false is returned
func TestIsAuthProxyReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			name: "Test IsReady when AuthProxy is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      ComponentName,
						Labels:    map[string]string{"app": ComponentName},
					},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": ComponentName},
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
						Name:      ComponentName + "-95d8c5d96-m6mbr",
						Labels: map[string]string{
							"pod-template-hash": "95d8c5d96",
							"app":               ComponentName,
						},
					},
				},
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   ComponentNamespace,
						Name:        ComponentName + "-95d8c5d96",
						Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
					},
				},
			).Build(),
			expectTrue: true,
		},
		{
			name: "Test IsReady when AuthProxy deployment is not ready",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      ComponentName,
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
						Replicas:          1,
						UpdatedReplicas:   0,
					},
				}).Build(),
			expectTrue: false,
		},
		{
			name:       "Test IsReady when AuthProxy deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	authProxy := NewComponent().(authProxyComponent)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			assert.Equal(t, tt.expectTrue, authProxy.isAuthProxyReady(ctx))
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
			name:         "DefaultConfig",
			description:  "Test default configuration of AuthProxy with no overrides",
			expectedYAML: "testdata/noOverrideValues.yaml",
			actualCR:     "testdata/noOverrideVz.yaml",
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "OverrideReplicas",
			description:  "Test override of replica count",
			expectedYAML: "testdata/replicasOverrideValues.yaml",
			actualCR:     "testdata/replicasOverrideVz.yaml",
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "OverrideAffinity",
			description:  "Test override of affinity configuration for AuthProxy",
			expectedYAML: "testdata/affinityOverrideValues.yaml",
			actualCR:     "testdata/affinityOverrideVz.yaml",
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "OverrideDNSWildcardDomain",
			description:  "Test overriding DNS wildcard domain",
			expectedYAML: "testdata/dnsWildcardDomainOverrideValues.yaml",
			actualCR:     "testdata/dnsWildcardDomainOverrideVz.yaml",
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "DisableAuthProxy",
			description:  "Test overriding AuthProxy to be disabled",
			expectedYAML: "testdata/enabledOverrideValues.yaml",
			actualCR:     "testdata/enabledOverrideVz.yaml",
			numKeyValues: 1,
			expectedErr:  nil,
		},
		{
			name:         "loadImageSettingsFailed",
			description:  "Test AppendOverrides when loadImageSettings get failed",
			expectedYAML: "testdata/enabledOverrideValues.yaml",
			actualCR:     "testdata/enabledOverrideVz.yaml",
			numKeyValues: 1,
			expectedErr:  fmt.Errorf("path error"),
		},
	}
	defer resetWriteFileFunc()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			asserts := assert.New(t)
			t.Log(test.description)

			// Read the Verrazzano CR into a struct
			testCR := vzapi.Verrazzano{}
			yamlFile, err := os.ReadFile(test.actualCR)
			asserts.NoError(err)
			err = yaml.Unmarshal(yamlFile, &testCR)
			asserts.NoError(err)

			fakeClient := createFakeClientWithIngress()
			fakeContext := spi.NewFakeContext(fakeClient, &testCR, nil, false, profileDir)

			setWriteFileFunc(test.expectedErr, asserts, test.expectedYAML)
			if test.name == "loadImageSettingsFailed" {
				// setting invalid bom file path for loadImageSettingsFailed test case
				config.SetDefaultBomFilePath("test")
			}
			var kvs []bom.KeyValue
			kvs, err = AppendOverrides(fakeContext, "", "", "", kvs)
			if test.expectedErr != nil {
				asserts.Error(err)
				asserts.Equal([]bom.KeyValue(nil), kvs)
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
	files, err := os.ReadDir(os.TempDir())
	assert.NoError(t, err, "Error reading temp dir to verify file cleanup")
	for _, file := range files {
		assert.False(t,
			strings.HasPrefix(file.Name(), tmpFilePrefix) && strings.HasSuffix(file.Name(), ".yaml"),
			"Found unexpected temp file remaining: %s", file.Name())
	}

}

// TestRemoveResourcePolicyAnnotation tests the removeResourcePolicyAnnotation function
// GIVEN a call to removeResourcePolicyAnnotation
//
//	WHEN I call with a object that is annotated with the resource policy annotation
//	THEN the annotation is removed
func TestRemoveResourcePolicyAnnotation(t *testing.T) {
	namespacedName := types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	obj := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
			Annotations: map[string]string{"meta.helm.sh/release-name": ComponentName, "meta.helm.sh/release-namespace": ComponentNamespace,
				"helm.sh/resource-policy": "keep"},
		},
	}

	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(obj).Build()
	res, err := common.RemoveResourcePolicyAnnotation(c, obj, namespacedName)
	assert.NoError(t, err)
	assert.Equal(t, ComponentName, res.GetAnnotations()["meta.helm.sh/release-name"])
	assert.Equal(t, globalconst.VerrazzanoSystemNamespace, res.GetAnnotations()["meta.helm.sh/release-namespace"])
	_, ok := res.GetAnnotations()["helm.sh/resource-policy"]
	assert.False(t, ok)
}

// TestAppendOverridesIfManagedCluster tests the AppendOverrides function when executed in a registered managed cluster
// GIVEN a call to AppendOverrides
//
//	WHEN I call the method when a registration secret is present
//	THEN the proxy override for client ID is set
func TestAppendOverridesIfManagedCluster(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vpoconst.MCRegistrationSecret,
				Namespace: vpoconst.VerrazzanoSystemNamespace,
			},
			Data: map[string][]byte{vpoconst.ClusterNameData: []byte("managed1")},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: vpoconst.NGINXControllerServiceName, Namespace: nginxutil.IngressNGINXNamespace()},
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
	fakeContext := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false, profileDir)

	var kvs []bom.KeyValue
	kvs, err := AppendOverrides(fakeContext, "", "", "", kvs)
	assert.NoError(t, err)

	data, err := os.ReadFile(kvs[0].Value)
	assert.NoError(t, err)
	overrides := &authProxyValues{}
	err = yaml.Unmarshal(data, overrides)
	assert.NoError(t, err)
	assert.Equal(t, "verrazzano-managed1", overrides.Proxy.OIDCClientID, "wrong client ID")
	assert.Equal(t, "verrazzano-pkce", overrides.Proxy.PKCEClientID, "wrong client ID")
}

// TestUninstallResources tests the authproxy Uninstall call
// GIVEN a authproxy component
//
//	WHEN I call Uninstall with the authproxy helm chart not installed
//	THEN ensure that all authproxy resources are explicitly deleted
func TestUninstallResources(t *testing.T) {
	defer helmcli.SetDefaultActionConfigFunction()
	helmcli.SetActionConfigFunction(testActionConfigWithUninstalledAuthproxy)

	clusterRole := &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: "impersonate-api-user"}}
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "impersonate-api-user"}}
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: "verrazzano-authproxy-config"}}
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: ComponentName}}
	ingress := &netv1.Ingress{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: "verrazzano-ingress"}}
	secret := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: "verrazzano-authproxy-secret"}}
	service1 := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: ComponentName}}
	service2 := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: "verrazzano-authproxy-elasticsearch"}}
	serviceAccount := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: "impersonate-api-user"}}

	c := fake.NewClientBuilder().WithScheme(clientgoscheme.Scheme).WithObjects(
		clusterRole,
		clusterRoleBinding,
		configMap,
		deployment,
		ingress,
		secret,
		service1,
		service2,
		serviceAccount,
	).Build()

	err := NewComponent().Uninstall(spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)

	// Assert that the resources have been deleted
	err = c.Get(context.TODO(), types.NamespacedName{Name: ComponentName}, clusterRole)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Name: ComponentName}, clusterRoleBinding)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, configMap)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, deployment)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, ingress)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, secret)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, service1)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, service2)
	assert.True(t, errors.IsNotFound(err))
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, serviceAccount)
	assert.True(t, errors.IsNotFound(err))
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
						AuthProxy: &vzapi.AuthProxyComponent{
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
						AuthProxy: &v1beta1.AuthProxyComponent{
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

func createFakeClientWithIngress() client.Client {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: vpoconst.NGINXControllerServiceName, Namespace: nginxutil.IngressNGINXNamespace()},
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

// cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	if err := vzos.RemoveTempFiles(ctx.Log().GetZapLogger(), tmpFileCleanPattern); err != nil {
		ctx.Log().Errorf("Failed deleting temp files: %v", err)
	}
}

// setWriteFileFunc sets the writeFileFunc variable for the test cases.
func setWriteFileFunc(expectedErr error, asserts *assert.Assertions, expectedYAML string) {
	writeFileFunc = func(filename string, data []byte, perm fs.FileMode) error {
		if expectedErr != nil {
			return expectedErr
		}
		if err := os.WriteFile(filename, data, perm); err != nil {
			asserts.Failf("Failure writing file %s: %s", filename, err)
			return err
		}
		asserts.FileExists(filename)

		// Unmarshal the actual generated helm values from code under test
		actualValues := authProxyValues{}
		err := yaml.Unmarshal(data, &actualValues)
		asserts.NoError(err)

		// read in the expected results' data from a file and unmarshal it into a values object
		expectedData, err := os.ReadFile(expectedYAML)
		asserts.NoError(err, "Error reading expected values yaml file %s", expectedYAML)
		expectedValues := authProxyValues{}
		err = yaml.Unmarshal(expectedData, &expectedValues)
		asserts.NoError(err)

		// Compare the actual and expected values objects
		asserts.Equal(expectedValues, actualValues)
		return nil
	}
}

// TestLoadImageSettings test the loadImageSettings to check images specified are loaded successfully
// GIVEN a call to  loadImageSettings
// WHEN I call loadImageSettings with valid BOM file with valid path
// THEN it loads the override values for the image name and version
//
// WHEN I call loadImageSettings with invalid BOM file or invalid path
// THEN error is returned
func TestLoadImageSettings(t *testing.T) {
	wantError := true
	getAuthProxyValue := func() *authProxyValues { return &authProxyValues{} }
	fakeContext := spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false)
	tests := []struct {
		name                string
		ctx                 spi.ComponentContext
		overrides           *authProxyValues
		overrideBomPathFunc func()
		wantErr             bool
	}{
		{
			"IncorrectBOMFilePathTest",
			fakeContext,
			getAuthProxyValue(),
			func() { config.SetDefaultBomFilePath("test") },
			wantError,
		},
		{
			"NoSubComponentBOMFileTest",
			fakeContext,
			getAuthProxyValue(),
			func() { config.SetDefaultBomFilePath("testdata/noSubComponentTestBOM.json") },
			wantError,
		},
		{
			"NoImageNameBOMFileTest",
			fakeContext,
			getAuthProxyValue(),
			func() { config.SetDefaultBomFilePath("testdata/noImageNameTestBOM.json") },
			wantError,
		},
		{
			"NoImageVersionBOMFileTest",
			fakeContext,
			getAuthProxyValue(),
			func() { config.SetDefaultBomFilePath("testdata/noImageVersionTestBOM.json") },
			wantError,
		},
		{
			"NoMetricsImageBOMFileTest",
			fakeContext,
			getAuthProxyValue(),
			func() { config.SetDefaultBomFilePath("testdata/noMetricsImageTestBOM.json") },
			wantError,
		},
		{
			"NoMetricsImageVersionBOMFileTest",
			fakeContext,
			&authProxyValues{},
			func() { config.SetDefaultBomFilePath("testdata/noMetricsImageVersionTestBOM.json") },
			wantError,
		},
	}
	defer config.SetDefaultBomFilePath("")
	for _, tt := range tests {
		tt.overrideBomPathFunc()
		t.Run(tt.name, func(t *testing.T) {
			if err := loadImageSettings(tt.ctx, tt.overrides); (err != nil) != tt.wantErr {
				t.Errorf("loadImageSettings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGetWildcardDNS test the getWildcardDNS
// GIVEN a call to  getWildcardDNS
// WHEN I call getWildcardDNS with DNS available Verrazzano Spec
// THEN it returns the Wildcard DNS or false in case Wildcard DNS is not available
func TestGetWildcardDNS(t *testing.T) {

	tests := []struct {
		name  string
		vz    *vzapi.VerrazzanoSpec
		want  bool
		want1 string
	}{
		{
			"test getWildcardDNS when wildcardDns is passed in Verrazzano spec",
			&vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					DNS: &vzapi.DNSComponent{
						Wildcard: &vzapi.Wildcard{Domain: "nip.io"},
					},
				},
			},
			true,
			"nip.io",
		},
		{
			"test getWildcardDNS when no DNS  is specified in Verrazzano spec",
			&vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{},
			},
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getWildcardDNS(tt.vz)
			if got != tt.want {
				t.Errorf("getWildcardDNS() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getWildcardDNS() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
