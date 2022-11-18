// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package ready

import (
	"testing"
	"time"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getScheme() *runtime.Scheme {
	var testScheme = runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(testScheme)

	_ = vzapi.AddToScheme(testScheme)
	_ = clustersv1alpha1.AddToScheme(testScheme)

	_ = certv1.AddToScheme(testScheme)
	return testScheme
}

// TestCheckCertificatesReady Tests the CertificatesAreReady func
// GIVEN a Verrazzano instance with CertManager enabled
// WHEN I call CertificatesAreReady with a list of cert names where both are ready
// THEN false and an empty list of names is returned
func TestCheckCertificatesReady(t *testing.T) {

	certNames := []types.NamespacedName{
		{Name: "mycert", Namespace: "verrazzano-system"},
		{Name: "mycert2", Namespace: "verrazzano-system"},
	}
	cmEnabled := true
	vz := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Namespace: "foo"},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				CertManager: &v1alpha1.CertManagerComponent{Enabled: &cmEnabled},
			},
		},
	}

	now := time.Now()
	time1 := metav1.NewTime(now.Add(-300 * time.Second))
	time2 := metav1.NewTime(now.Add(-180 * time.Second))
	time3 := metav1.NewTime(now)

	client := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		&certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certNames[0].Name, Namespace: certNames[0].Namespace},
			Spec:       certv1.CertificateSpec{},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionIssuing, Status: cmmeta.ConditionUnknown, LastTransitionTime: &time1},
					{Type: certv1.CertificateConditionIssuing, Status: cmmeta.ConditionFalse, LastTransitionTime: &time2},
					{Type: certv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time3},
				},
			},
		},
		&certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certNames[1].Name, Namespace: certNames[1].Namespace},
			Spec:       certv1.CertificateSpec{},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time3},
				},
			},
		},
	).Build()
	allReady, notReadyCerts := CertificatesAreReady(client, vzlog.DefaultLogger(), vz, certNames)
	assert.True(t, allReady)
	assert.Len(t, notReadyCerts, 0)
}

// TestCheckCertificatesNotReady Tests the CertificatesAreReady func
// GIVEN a Verrazzano instance with CertManager enabled
// WHEN I call CertificatesAreReady with a list of cert names where one is ready and one isn't
// THEN false and the returned list of names has the name of the cert that isn't ready
func TestCheckCertificatesNotReady(t *testing.T) {

	certNames := []types.NamespacedName{
		{Name: "mycert", Namespace: "verrazzano-system"},
		{Name: "mycert2", Namespace: "verrazzano-system"},
	}
	notReadyExpected := []types.NamespacedName{
		certNames[1],
	}
	cmEnabled := true
	vz := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Namespace: "foo"},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				CertManager: &v1alpha1.CertManagerComponent{Enabled: &cmEnabled},
			},
		},
	}

	now := time.Now()
	time1 := metav1.NewTime(now.Add(-300 * time.Second))
	time3 := metav1.NewTime(now)

	client := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		&certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certNames[0].Name, Namespace: certNames[0].Namespace},
			Spec:       certv1.CertificateSpec{},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionIssuing, Status: cmmeta.ConditionUnknown, LastTransitionTime: &time1},
					{Type: certv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time3},
				},
			},
		},
		&certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{Name: certNames[1].Name, Namespace: certNames[1].Namespace},
			Spec:       certv1.CertificateSpec{},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionIssuing, Status: cmmeta.ConditionFalse, LastTransitionTime: &time3},
				},
			},
		},
	).Build()
	allReady, notReadyActual := CertificatesAreReady(client, vzlog.DefaultLogger(), vz, certNames)
	assert.False(t, allReady)
	assert.Equal(t, notReadyExpected, notReadyActual)
}

// TestCheckCertificatesNotReadyCertManagerDisabled Tests the CertificatesAreReady func
// GIVEN a Verrazzano instance with CertManager disabled
// WHEN I call CertificatesAreReady with a non-empty certs list
// THEN true and an empty list of names is returned
func TestCheckCertificatesNotReadyCertManagerDisabled(t *testing.T) {
	certNames := []types.NamespacedName{
		{Name: "mycert", Namespace: "verrazzano-system"},
		{Name: "mycert2", Namespace: "verrazzano-system"},
	}

	cmEnabled := false
	vz := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Namespace: "foo"},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				CertManager: &v1alpha1.CertManagerComponent{Enabled: &cmEnabled},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	allReady, notReadyActual := CertificatesAreReady(client, vzlog.DefaultLogger(), vz, certNames)
	assert.True(t, allReady)
	assert.Len(t, notReadyActual, 0)
}

// TestCheckCertificatesNotReadyNoCertsPassed Tests the CertificatesAreReady func
// GIVEN a Verrazzano instance with CertManager enabled
// WHEN I call CertificatesAreReady with an empty certs list
// THEN true and an empty list of names is returned
func TestCheckCertificatesNotReadyNoCertsPassed(t *testing.T) {
	cmEnabled := true
	vz := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Namespace: "foo"},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				CertManager: &v1alpha1.CertManagerComponent{Enabled: &cmEnabled},
			},
		},
	}

	client := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	allReady, notReady := CertificatesAreReady(client, vzlog.DefaultLogger(), vz, []types.NamespacedName{})
	assert.Len(t, notReady, 0)
	assert.True(t, allReady)
}
