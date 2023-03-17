// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package opensearchoperator

import (
	"context"
	"fmt"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certv1fake "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzclusters "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	spi2 "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			OpenSearchOperator: &vzapi.OpenSearchOperatorComponent{Enabled: getBoolPtr(true)},
		},
	},
}

const (
	testNamespace   = "testNamespace"
	testBomFile     = "../../testdata/test_bom.json"
	fooDomainSuffix = "foo.com"
	profileDir      = "../../../../manifests/profiles"
)

// default CA object
var ca = vzapi.CA{
	SecretName:               "testSecret",
	ClusterResourceNamespace: testNamespace,
}

// Default Verrazzano object
var defaultVZConfig = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "myenv",
		Components: vzapi.ComponentSpec{
			CertManager: &vzapi.CertManagerComponent{
				Certificate: vzapi.Certificate{},
			},
		},
	},
}
var (
	testScheme = runtime.NewScheme()
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

// TestComponentNamespace tests the OpenSearchDashboard ComponentNamespace call
// GIVEN an OpenSearchDashboard component
//
//	WHEN I call ComponentNamespace with defaults
//	THEN the component namespace is returned
func TestComponentNamespace(t *testing.T) {
	NameSpace := NewComponent().Namespace()
	assert.Equal(t, NameSpace, constants.VerrazzanoLoggingNamespace)

}

// TestPostInstall tests the OpenSearch-Dashboards PostInstall call
// GIVEN an OpenSearch-Dashboards component
//
//	WHEN I call PostInstall
//	THEN no error is returned
func TestPostInstall(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				OpenSearchOperator: &vzapi.OpenSearchOperatorComponent{Enabled: getBoolPtr(true)},
			},
		},
	}, nil, false)
	vzComp := NewComponent()
	// PostInstall will fail because the expected VZ ingresses are not present in cluster
	result := certv1.ClusterIssuer{}
	result.Name = "verrazzano-cluster-issuer"
	result.Kind = "ClusterIssuer"
	//certv1fake.NewSimpleClientset()
	client := certv1fake.NewSimpleClientset()
	res, err1 := client.CertmanagerV1().ClusterIssuers().Create(context.TODO(), &result, metav1.CreateOptions{})
	fmt.Println(res)
	if err1 != nil {
		fmt.Errorf("error...")
	}
	err := vzComp.PostInstall(ctx)
	assert.IsType(t, spi2.RetryableError{}, err)
}

func getBoolPtr(b bool) *bool {
	return &b
}
