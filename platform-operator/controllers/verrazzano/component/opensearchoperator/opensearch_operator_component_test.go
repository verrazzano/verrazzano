// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package opensearchoperator

import (
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzclusters "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"testing"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	spi2 "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			OpenSearchOperator: &vzapi.OpenSearchOperatorComponent{Enabled: getBoolPtr(true)},
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
	err := vzComp.PostInstall(ctx)
	assert.IsType(t, spi2.RetryableError{}, err)

	// now get all the ingresses for VZ and add them to the fake K8S and ensure that PostInstall succeeds
	// when all the ingresses are present in the cluster
	err = vzComp.PostInstall(ctx)
	assert.NoError(t, err)
}

func getBoolPtr(b bool) *bool {
	return &b
}
