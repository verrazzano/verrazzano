// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	//"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)

	_ = v1alpha1.AddToScheme(testScheme)
	_ = clustersv1alpha1.AddToScheme(testScheme)

	_ = istioclinet.AddToScheme(testScheme)
	_ = istioclisec.AddToScheme(testScheme)
	_ = certv1.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// TestContextProfilesMerge Tests the profiles context merge
// GIVEN a Verrazzano instance with a profile
// WHEN I call NewContext
// THEN the correct correct context is created with the proper merge of the profile and user overrides
func TestContextProfilesMerge(t *testing.T) {
	config.TestProfilesDir = "../../../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	tests := []struct {
		name         string
		description  string
		expectedYAML string
		actualCR     v1alpha1.Verrazzano
		expectedErr  bool
	}{
		{
			name:         "TestBasicDevProfileWithStatus",
			description:  "Tests basic dev profile overrides",
			actualCR:     basicDevWithStatus,
			expectedYAML: basicDevMerged,
		},
		{
			name:         "TestBasicProdProfileWithStatus",
			description:  "Tests basic prod profile overrides",
			actualCR:     basicProdWithStatus,
			expectedYAML: basicProdMerged,
		},
		{
			name:         "TestBasicManagedClusterProfileWithStatus",
			description:  "Tests basic managed-cluster profile overrides",
			actualCR:     basicMgdClusterWithStatus,
			expectedYAML: basicManagedClusterMerged,
		},
		{
			name:         "TestBasicDevAllDisabled",
			description:  "Tests dev profile with all components disabled",
			actualCR:     devAllDisabledOverride,
			expectedYAML: devAllDisabledMerged,
		},
		{
			name:         "TestDevProfileOCIDNSOverride",
			description:  "Tests dev profile with OCI DNS overrides",
			actualCR:     devOCIDNSOverride,
			expectedYAML: devOCIDNSOverrideMerged,
		},
		{
			name:         "TestDevProfileCertManagerNoCert",
			description:  "Tests dev profile with Cert-Manager with no certificate",
			actualCR:     devCertManagerNoCert,
			expectedYAML: basicDevMerged,
		},
		{
			name:         "TestDevProfileCertManagerOverride",
			description:  "Tests dev profile with Cert-Manager overrides",
			actualCR:     devCertManagerOverride,
			expectedYAML: devCertManagerOverrideMerged,
		},
		{
			name:         "TestDevProfileElasticsearchOverrides",
			description:  "Tests dev profile with Elasticsearch installArg and persistence overrides",
			actualCR:     devElasticSearchOverrides,
			expectedYAML: devElasticSearchOveridesMerged,
		},
		{
			name:         "TestDevProfileKeycloakOverrides",
			description:  "Tests dev profile with Keycloak/MySQL installArg and persistence overrides",
			actualCR:     devKeycloakOverrides,
			expectedYAML: devKeycloakOveridesMerged,
		},
		{
			name:         "TestProdProfileElasticsearchOverrides",
			description:  "Tests prod profile with Elasticsearch installArg and persistence overrides",
			actualCR:     prodElasticSearchOverrides,
			expectedYAML: prodElasticSearchOveridesMerged,
		},
		{
			name:         "TestProdProfileElasticsearchStorageArgs",
			description:  "Tests prod profile with Elasticsearch storage installArgs",
			actualCR:     prodElasticSearchStorageArgs,
			expectedYAML: prodElasticSearchStorageMerged,
		},
		{
			name:         "TestProdProfileIngressIstioOverrides",
			description:  "Test prod profile with Istio and NGINX Ingress overrides",
			actualCR:     prodIngressIstioOverrides,
			expectedYAML: prodIngressIstioOverridesMerged,
		},
		{
			name:         "TestProdProfileFluentdOverrides",
			description:  "Test prod profile with Fluentd overrides",
			actualCR:     prodFluentdOverrides,
			expectedYAML: prodFluentdOverridesMerged,
		},
		{
			name:         "TestManagedClusterEnableAllOverrides",
			description:  "Test managed-cluster profile with overrides to enable everything",
			actualCR:     managedClusterEnableAllOverride,
			expectedYAML: managedClusterEnableAllMerged,
		},
		{
			name:         "TestProdNoStorageOpenSearchOverrides",
			description:  "Test prod profile with no storage and OpenSearch overrides",
			actualCR:     prodNoStorageOSOverrides,
			expectedYAML: prodNoStorageOpenSearchOverrides,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)

			t.Log(test.description)

			// Load the expected merged result into a VZ CR
			expectedVZ, err := loadExpectedMergeResult(test.expectedYAML)
			a.NoError(err)
			a.NotNil(expectedVZ)

			// Create the context with the effective CR
			log := vzlog.DefaultLogger()
			context, err := NewContext(log, fake.NewClientBuilder().WithScheme(testScheme).Build(), &test.actualCR, nil, false)
			// Assert the error expectations
			if test.expectedErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}

			a.NotNil(context, "Context was nil")
			a.NotNil(context.ActualCR(), "Actual CR was nil")
			a.Equal(test.actualCR, *context.ActualCR(), "Actual CR unexpectedly modified")
			a.NotNil(context.EffectiveCR(), "Effective CR was nil")
			a.Equal(v1alpha1.VerrazzanoStatus{}, context.EffectiveCR().Status, "Effective CR status not empty")
			a.True(equality.Semantic.DeepEqual(expectedVZ, context.EffectiveCR()), "Effective CR did not match expected results")
			a.Equal(log, context.Log(), "The log in the context doesn't match the original one")
		})
	}
}

func loadExpectedMergeResult(expectedYamlFile string) (*v1alpha1.Verrazzano, error) {
	bYaml, err := os.ReadFile(filepath.Join(expectedYamlFile))
	if err != nil {
		return nil, err
	}
	vz := v1alpha1.Verrazzano{}
	err = yaml.Unmarshal(bYaml, &vz)
	return &vz, err
}

func TestGetFakeContext(t *testing.T) {
	tests := []struct {
		name         string
		description  string
		expectedYAML string
		actualCR     v1alpha1.Verrazzano
		expectedErr  bool
	}{
		{
			name:         "TestBasicDevProfileWithStatus",
			description:  "Tests basic dev profile overrides",
			actualCR:     basicDevWithStatus,
			expectedYAML: basicDevMerged,
		},
		{
			name:         "TestBasicProdProfileWithStatus",
			description:  "Tests basic prod profile overrides",
			actualCR:     basicProdWithStatus,
			expectedYAML: basicProdMerged,
		},
		{
			name:         "TestBasicManagedClusterProfileWithStatus",
			description:  "Tests basic managed-cluster profile overrides",
			actualCR:     basicMgdClusterWithStatus,
			expectedYAML: basicManagedClusterMerged,
		},
		{
			name:         "TestBasicDevAllDisabled",
			description:  "Tests dev profile with all components disabled",
			actualCR:     devAllDisabledOverride,
			expectedYAML: devAllDisabledMerged,
		},
		{
			name:         "TestDevProfileOCIDNSOverride",
			description:  "Tests dev profile with OCI DNS overrides",
			actualCR:     devOCIDNSOverride,
			expectedYAML: devOCIDNSOverrideMerged,
		},
		{
			name:         "TestDevProfileCertManagerNoCert",
			description:  "Tests dev profile with Cert-Manager with no certificate",
			actualCR:     devCertManagerNoCert,
			expectedYAML: basicDevMerged,
		},
		{
			name:         "TestDevProfileCertManagerOverride",
			description:  "Tests dev profile with Cert-Manager overrides",
			actualCR:     devCertManagerOverride,
			expectedYAML: devCertManagerOverrideMerged,
		},
		{
			name:         "TestDevProfileElasticsearchOverrides",
			description:  "Tests dev profile with Elasticsearch installArg and persistence overrides",
			actualCR:     devElasticSearchOverrides,
			expectedYAML: devElasticSearchOveridesMerged,
		},
		{
			name:         "TestDevProfileKeycloakOverrides",
			description:  "Tests dev profile with Keycloak/MySQL installArg and persistence overrides",
			actualCR:     devKeycloakOverrides,
			expectedYAML: devKeycloakOveridesMerged,
		},
		{
			name:         "TestProdProfileElasticsearchOverrides",
			description:  "Tests prod profile with Elasticsearch installArg and persistence overrides",
			actualCR:     prodElasticSearchOverrides,
			expectedYAML: prodElasticSearchOveridesMerged,
		},
		{
			name:         "TestProdProfileElasticsearchStorageArgs",
			description:  "Tests prod profile with Elasticsearch storage installArgs",
			actualCR:     prodElasticSearchStorageArgs,
			expectedYAML: prodElasticSearchStorageMerged,
		},
		{
			name:         "TestProdProfileIngressIstioOverrides",
			description:  "Test prod profile with Istio and NGINX Ingress overrides",
			actualCR:     prodIngressIstioOverrides,
			expectedYAML: prodIngressIstioOverridesMerged,
		},
		{
			name:         "TestProdProfileFluentdOverrides",
			description:  "Test prod profile with Fluentd overrides",
			actualCR:     prodFluentdOverrides,
			expectedYAML: prodFluentdOverridesMerged,
		},
		{
			name:         "TestManagedClusterEnableAllOverrides",
			description:  "Test managed-cluster profile with overrides to enable everything",
			actualCR:     managedClusterEnableAllOverride,
			expectedYAML: managedClusterEnableAllMerged,
		},
		{
			name:         "TestProdNoStorageOpenSearchOverrides",
			description:  "Test prod profile with no storage and OpenSearch overrides",
			actualCR:     prodNoStorageOSOverrides,
			expectedYAML: prodNoStorageOpenSearchOverrides,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)

			t.Log(test.description)

			expectedVZ, err := loadExpectedMergeResult(test.expectedYAML)
			a.NoError(err)
			a.NotNil(expectedVZ)

			testArr := []string{"../../../../manifests/profiles"}
			client := fake.NewClientBuilder().WithScheme(testScheme).Build()
			context := NewFakeContext(client, &test.actualCR, nil, false, testArr...)

			a.NotNil(context, "Context was nil")
			a.NotNil(context.ActualCR(), "Actual CR was nil")
			a.Equal(test.actualCR, *context.ActualCR(), "Actual CR unexpectedly modified")
			a.NotNil(context.EffectiveCR(), "Effective CR was nil")
			a.Equal(v1alpha1.VerrazzanoStatus{}, context.EffectiveCR().Status, "Effective CR status not empty")
			a.True(equality.Semantic.DeepEqual(expectedVZ, context.EffectiveCR()), "Effective CR did not match expected results")
			a.Equal(client, context.Client(), "The client name is incorrect")
			a.Equal(context.IsDryRun(), false, "The dryRun value is incorrect")
			//a.Equal(context.ActualCRV1Beta1(), *v1beta1.Verrazzano((*v1beta1.Verrazzano)(nil)) ,"The v1beta1 CR value is incorrect")
		})
	}

}

func TestOperation(t *testing.T) {
	config.TestProfilesDir = "../../../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	tests := []struct {
		name         string
		description  string
		expectedYAML string
		actualCR     v1alpha1.Verrazzano
		expectedErr  bool
	}{
		{
			name:         "TestBasicDevProfileWithStatus",
			description:  "Tests basic dev profile overrides",
			actualCR:     basicDevWithStatus,
			expectedYAML: basicDevMerged,
		},
		{
			name:         "TestBasicProdProfileWithStatus",
			description:  "Tests basic prod profile overrides",
			actualCR:     basicProdWithStatus,
			expectedYAML: basicProdMerged,
		},
		{
			name:         "TestBasicManagedClusterProfileWithStatus",
			description:  "Tests basic managed-cluster profile overrides",
			actualCR:     basicMgdClusterWithStatus,
			expectedYAML: basicManagedClusterMerged,
		},
		{
			name:         "TestBasicDevAllDisabled",
			description:  "Tests dev profile with all components disabled",
			actualCR:     devAllDisabledOverride,
			expectedYAML: devAllDisabledMerged,
		},
		{
			name:         "TestDevProfileOCIDNSOverride",
			description:  "Tests dev profile with OCI DNS overrides",
			actualCR:     devOCIDNSOverride,
			expectedYAML: devOCIDNSOverrideMerged,
		},
		{
			name:         "TestDevProfileCertManagerNoCert",
			description:  "Tests dev profile with Cert-Manager with no certificate",
			actualCR:     devCertManagerNoCert,
			expectedYAML: basicDevMerged,
		},
		{
			name:         "TestDevProfileCertManagerOverride",
			description:  "Tests dev profile with Cert-Manager overrides",
			actualCR:     devCertManagerOverride,
			expectedYAML: devCertManagerOverrideMerged,
		},
		{
			name:         "TestDevProfileElasticsearchOverrides",
			description:  "Tests dev profile with Elasticsearch installArg and persistence overrides",
			actualCR:     devElasticSearchOverrides,
			expectedYAML: devElasticSearchOveridesMerged,
		},
		{
			name:         "TestDevProfileKeycloakOverrides",
			description:  "Tests dev profile with Keycloak/MySQL installArg and persistence overrides",
			actualCR:     devKeycloakOverrides,
			expectedYAML: devKeycloakOveridesMerged,
		},
		{
			name:         "TestProdProfileElasticsearchOverrides",
			description:  "Tests prod profile with Elasticsearch installArg and persistence overrides",
			actualCR:     prodElasticSearchOverrides,
			expectedYAML: prodElasticSearchOveridesMerged,
		},
		{
			name:         "TestProdProfileElasticsearchStorageArgs",
			description:  "Tests prod profile with Elasticsearch storage installArgs",
			actualCR:     prodElasticSearchStorageArgs,
			expectedYAML: prodElasticSearchStorageMerged,
		},
		{
			name:         "TestProdProfileIngressIstioOverrides",
			description:  "Test prod profile with Istio and NGINX Ingress overrides",
			actualCR:     prodIngressIstioOverrides,
			expectedYAML: prodIngressIstioOverridesMerged,
		},
		{
			name:         "TestProdProfileFluentdOverrides",
			description:  "Test prod profile with Fluentd overrides",
			actualCR:     prodFluentdOverrides,
			expectedYAML: prodFluentdOverridesMerged,
		},
		{
			name:         "TestManagedClusterEnableAllOverrides",
			description:  "Test managed-cluster profile with overrides to enable everything",
			actualCR:     managedClusterEnableAllOverride,
			expectedYAML: managedClusterEnableAllMerged,
		},
		{
			name:         "TestProdNoStorageOpenSearchOverrides",
			description:  "Test prod profile with no storage and OpenSearch overrides",
			actualCR:     prodNoStorageOSOverrides,
			expectedYAML: prodNoStorageOpenSearchOverrides,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)

			t.Log(test.description)

			// Load the expected merged result into a VZ CR
			expectedVZ, err := loadExpectedMergeResult(test.expectedYAML)
			a.NoError(err)
			a.NotNil(expectedVZ)

			// Create the context with the effective CR
			log := vzlog.DefaultLogger()
			context, err := NewContext(log, fake.NewClientBuilder().WithScheme(testScheme).Build(), &test.actualCR, nil, false)
			// Assert the error expectations
			if test.expectedErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
			contextNew := context.Operation("foo")
			a.Equal(contextNew.GetOperation(), "foo", "The operation is incorrect")

		})
	}

}

func TestInitAndCopy(t *testing.T) {
	config.TestProfilesDir = "../../../../manifests/profiles"
	defer func() { config.TestProfilesDir = "" }()

	tests := []struct {
		name         string
		description  string
		expectedYAML string
		actualCR     v1alpha1.Verrazzano
		expectedErr  bool
	}{
		{
			name:         "TestBasicDevProfileWithStatus",
			description:  "Tests basic dev profile overrides",
			actualCR:     basicDevWithStatus,
			expectedYAML: basicDevMerged,
		},
		{
			name:         "TestBasicProdProfileWithStatus",
			description:  "Tests basic prod profile overrides",
			actualCR:     basicProdWithStatus,
			expectedYAML: basicProdMerged,
		},
		{
			name:         "TestBasicManagedClusterProfileWithStatus",
			description:  "Tests basic managed-cluster profile overrides",
			actualCR:     basicMgdClusterWithStatus,
			expectedYAML: basicManagedClusterMerged,
		},
		{
			name:         "TestBasicDevAllDisabled",
			description:  "Tests dev profile with all components disabled",
			actualCR:     devAllDisabledOverride,
			expectedYAML: devAllDisabledMerged,
		},
		{
			name:         "TestDevProfileOCIDNSOverride",
			description:  "Tests dev profile with OCI DNS overrides",
			actualCR:     devOCIDNSOverride,
			expectedYAML: devOCIDNSOverrideMerged,
		},
		{
			name:         "TestDevProfileCertManagerNoCert",
			description:  "Tests dev profile with Cert-Manager with no certificate",
			actualCR:     devCertManagerNoCert,
			expectedYAML: basicDevMerged,
		},
		{
			name:         "TestDevProfileCertManagerOverride",
			description:  "Tests dev profile with Cert-Manager overrides",
			actualCR:     devCertManagerOverride,
			expectedYAML: devCertManagerOverrideMerged,
		},
		{
			name:         "TestDevProfileElasticsearchOverrides",
			description:  "Tests dev profile with Elasticsearch installArg and persistence overrides",
			actualCR:     devElasticSearchOverrides,
			expectedYAML: devElasticSearchOveridesMerged,
		},
		{
			name:         "TestDevProfileKeycloakOverrides",
			description:  "Tests dev profile with Keycloak/MySQL installArg and persistence overrides",
			actualCR:     devKeycloakOverrides,
			expectedYAML: devKeycloakOveridesMerged,
		},
		{
			name:         "TestProdProfileElasticsearchOverrides",
			description:  "Tests prod profile with Elasticsearch installArg and persistence overrides",
			actualCR:     prodElasticSearchOverrides,
			expectedYAML: prodElasticSearchOveridesMerged,
		},
		{
			name:         "TestProdProfileElasticsearchStorageArgs",
			description:  "Tests prod profile with Elasticsearch storage installArgs",
			actualCR:     prodElasticSearchStorageArgs,
			expectedYAML: prodElasticSearchStorageMerged,
		},
		{
			name:         "TestProdProfileIngressIstioOverrides",
			description:  "Test prod profile with Istio and NGINX Ingress overrides",
			actualCR:     prodIngressIstioOverrides,
			expectedYAML: prodIngressIstioOverridesMerged,
		},
		{
			name:         "TestProdProfileFluentdOverrides",
			description:  "Test prod profile with Fluentd overrides",
			actualCR:     prodFluentdOverrides,
			expectedYAML: prodFluentdOverridesMerged,
		},
		{
			name:         "TestManagedClusterEnableAllOverrides",
			description:  "Test managed-cluster profile with overrides to enable everything",
			actualCR:     managedClusterEnableAllOverride,
			expectedYAML: managedClusterEnableAllMerged,
		},
		{
			name:         "TestProdNoStorageOpenSearchOverrides",
			description:  "Test prod profile with no storage and OpenSearch overrides",
			actualCR:     prodNoStorageOSOverrides,
			expectedYAML: prodNoStorageOpenSearchOverrides,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)

			t.Log(test.description)

			// Load the expected merged result into a VZ CR
			expectedVZ, err := loadExpectedMergeResult(test.expectedYAML)
			a.NoError(err)
			a.NotNil(expectedVZ)

			// Create the context with the effective CR
			log := vzlog.DefaultLogger()
			context, err := NewContext(log, fake.NewClientBuilder().WithScheme(testScheme).Build(), &test.actualCR, nil, false)
			// Assert the error expectations
			if test.expectedErr {
				a.Error(err)
			} else {
				a.NoError(err)
			}
			contextNew := context.Init("grafana")
			a.Equal(contextNew.GetComponent(), "grafana", "The component name is incorrect")
			//a.NotEqual(contextNew.GetComponent(),"fluentd","The component name should not have been same")

			contextMod := context.Copy()
			a.Equal(context, contextMod, "The two contexts don't match")
		})
	}

}
