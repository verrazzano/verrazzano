// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1alpha3"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)

	_ = vzapi.AddToScheme(testScheme)
	_ = clustersv1alpha1.AddToScheme(testScheme)

	_ = istioclinet.AddToScheme(testScheme)
	_ = istioclisec.AddToScheme(testScheme)
	_ = certapiv1.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// TestPatchArgoCDIngress should annotate the ArgoCD ingress with Acme/Private CA values
// GIVEN a ArgoCD Ingress and a Verrazzano CR
//
//	WHEN patchArgoCDIngress is called
//	THEN patchRArgoCDIngress should annotate the ingress according to the Verrazzano CR
func TestPatchArgoCDIngress(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: constants.ArgoCDIngress, Namespace: constants.ArgoCDNamespace},
	}
	time := metav1.Now()
	cert := &certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: certificates[0].Name, Namespace: certificates[0].Namespace},
		Status: certapiv1.CertificateStatus{
			Conditions: []certapiv1.CertificateCondition{
				{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(ingress, cert).Build()
	ctx := spi.NewFakeContext(fakeClient, vz, nil, false)
	assert.NoError(t, patchArgoCDIngress(ctx))
}
