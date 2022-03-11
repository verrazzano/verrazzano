// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	"testing"
	"time"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckCertificatesReady(t *testing.T) {

	certNames := []types.NamespacedName{
		{Name: "mycert", Namespace: "verrazzano-system"},
		{Name: "mycert2", Namespace: "verrazzano-system"},
	}
	trueVal := true
	vz := &v1alpha1.Verrazzano{
		ObjectMeta: v1.ObjectMeta{Namespace: "foo"},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				CertManager: &v1alpha1.CertManagerComponent{Enabled: &trueVal},
			},
		},
	}

	now := time.Now()
	time1 := metav1.NewTime(now.Add(-300 * time.Second))
	time2 := metav1.NewTime(now.Add(-180 * time.Second))
	time3 := metav1.NewTime(now)

	client := fake.NewFakeClientWithScheme(testScheme,
		&certv1.Certificate{
			ObjectMeta: v1.ObjectMeta{Name: certNames[0].Name, Namespace: certNames[0].Namespace},
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
			ObjectMeta: v1.ObjectMeta{Name: certNames[1].Name, Namespace: certNames[1].Namespace},
			Spec:       certv1.CertificateSpec{},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time3},
				},
			},
		},
	)
	ctx := NewFakeContext(client, vz, false)
	notReady := CheckCertificatesReady(ctx, certNames)
	assert.Len(t, notReady, 0)
}

func TestCheckCertificatesNotReady(t *testing.T) {

	certNames := []types.NamespacedName{
		{Name: "mycert", Namespace: "verrazzano-system"},
		{Name: "mycert2", Namespace: "verrazzano-system"},
	}
	trueVal := true
	vz := &v1alpha1.Verrazzano{
		ObjectMeta: v1.ObjectMeta{Namespace: "foo"},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				CertManager: &v1alpha1.CertManagerComponent{Enabled: &trueVal},
			},
		},
	}

	now := time.Now()
	time1 := metav1.NewTime(now.Add(-300 * time.Second))
	time3 := metav1.NewTime(now)

	client := fake.NewFakeClientWithScheme(testScheme,
		&certv1.Certificate{
			ObjectMeta: v1.ObjectMeta{Name: certNames[0].Name, Namespace: certNames[0].Namespace},
			Spec:       certv1.CertificateSpec{},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionIssuing, Status: cmmeta.ConditionUnknown, LastTransitionTime: &time1},
					{Type: certv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time3},
				},
			},
		},
		&certv1.Certificate{
			ObjectMeta: v1.ObjectMeta{Name: certNames[1].Name, Namespace: certNames[1].Namespace},
			Spec:       certv1.CertificateSpec{},
			Status: certv1.CertificateStatus{
				Conditions: []certv1.CertificateCondition{
					{Type: certv1.CertificateConditionIssuing, Status: cmmeta.ConditionFalse, LastTransitionTime: &time3},
				},
			},
		},
	)
	ctx := NewFakeContext(client, vz, false)
	notReady := CheckCertificatesReady(ctx, certNames)
	assert.Len(t, notReady, 1)
}
