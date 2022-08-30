// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
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
	config.TestProfilesDir = "../../../../manifests/profiles/v1alpha1"
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
			a.Equal(expectedVZ, context.EffectiveCR(), "Effective CR did not match expected results")
		})
	}
}

func loadExpectedMergeResult(expectedYamlFile string) (*v1alpha1.Verrazzano, error) {
	bYaml, err := ioutil.ReadFile(filepath.Join(expectedYamlFile))
	if err != nil {
		return nil, err
	}
	vz := v1alpha1.Verrazzano{}
	err = yaml.Unmarshal(bYaml, &vz)
	return &vz, err
}
