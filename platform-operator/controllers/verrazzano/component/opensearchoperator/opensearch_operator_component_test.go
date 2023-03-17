// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package opensearchoperator

import (
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certv1fake "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	certv1client "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzclusters "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
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
	testNamespace = "testNamespace"
)

// default CA object
var ca = vzapi.CA{
	SecretName:               "testSecret",
	ClusterResourceNamespace: testNamespace,
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
	//Create fake clusterIssuer
	clusterIssuer := &certv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "verrazzano-cluster-issuer"},
		Spec:   certv1.IssuerSpec{},
		Status: certv1.IssuerStatus{},
	}
	cmClient := certv1fake.NewSimpleClientset(clusterIssuer)
	defer func() { certmanager.GetCMClientFunc = certmanager.GetCertManagerClientset }()
	certmanager.GetCMClientFunc = func() (certv1client.CertmanagerV1Interface, error) {
		return cmClient.CertmanagerV1(), nil
	}
	err := vzComp.PostInstall(ctx)
	//There should be no error on post install, means certificates has been created.
	assert.IsType(t, nil, err)
}

func getBoolPtr(b bool) *bool {
	return &b
}
