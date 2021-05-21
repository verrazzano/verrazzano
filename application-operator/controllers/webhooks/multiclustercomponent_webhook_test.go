// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"github.com/stretchr/testify/assert"
	v1alpha12 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

// newMultiClusterComponentValidator creates a new MultiClusterComponentValidator
func newMultiClusterComponentValidator() MultiClusterComponentValidator {
	scheme := newScheme()
	decoder, _ := admission.NewDecoder(scheme)
	cli := fake.NewFakeClientWithScheme(scheme)
	v := MultiClusterComponentValidator{client: cli, decoder: decoder}
	return v
}

// TestValidationFailureForMultiClusterComponentCreationWithoutTargetClusters tests preventing the creation
// of a MultiClusterComponent resources that is missing Placement information.
// GIVEN a call to validate a MultiClusterComponent resource
// WHEN the MultiClusterComponent resource is missing Placement information
// THEN the validation should fail.
func TestValidationFailureForMultiClusterComponentCreationWithoutTargetClusters(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterComponentValidator()
	p := v1alpha12.MultiClusterComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mccomponent-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterComponentSpec{},
	}

	req := newAdmissionRequest(admissionv1beta1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster component validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "target cluster")

	req = newAdmissionRequest(admissionv1beta1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster component validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "target cluster")
}

// TestValidationFailureForMultiClusterComponentCreationTargetingMissingManagedCluster tests preventing the creation
// of a MultiClusterComponent resources that references a non-existent managed cluster.
// GIVEN a call to validate a MultiClusterComponent resource
// WHEN the MultiClusterComponent resource references a VerrazzanoManagedCluster that does not exist
// THEN the validation should fail.
func TestValidationFailureForMultiClusterComponentCreationTargetingMissingManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterComponentValidator()
	p := v1alpha12.MultiClusterComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mccomponent-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterComponentSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "invalid-cluster-name"}},
			},
		},
	}

	req := newAdmissionRequest(admissionv1beta1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster component validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "invalid-cluster-name")

	req = newAdmissionRequest(admissionv1beta1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster component validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "invalid-cluster-name")
}

// TestValidationSuccessForMultiClusterComponentCreationTargetingExistingManagedCluster tests allowing the creation
// of a MultiClusterComponent resources that references an existent managed cluster.
// GIVEN a call to validate a MultiClusterComponent resource
// WHEN the MultiClusterComponent resource references a VerrazzanoManagedCluster that does exist
// THEN the validation should pass.
func TestValidationSuccessForMultiClusterComponentCreationTargetingExistingManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterComponentValidator()
	mc := v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "valid-cluster-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha1.VerrazzanoManagedClusterSpec{
			PrometheusSecret:             "test-prometheus-secret",
			ManagedClusterManifestSecret: "test-cluster-manifest-secret",
			ServiceAccount:               "test-service-account",
		},
	}
	mcc := v1alpha12.MultiClusterComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mccomponent-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterComponentSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "valid-cluster-name"}},
			},
		},
	}
	asrt.NoError(v.client.Create(context.TODO(), &mc))

	req := newAdmissionRequest(admissionv1beta1.Create, mcc)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster component create validation to succeed.")

	req = newAdmissionRequest(admissionv1beta1.Update, mcc)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster component create validation to succeed.")
}

// TestValidationSuccessForMultiClusterComponentCreationWithoutTargetClustersOnManagedCluster tests allowing the creation
// of a MultiClusterComponent resources that is missing target cluster information when on managed cluster.
// GIVEN a call to validate a MultiClusterComponent resource
// WHEN the MultiClusterComponent resource is missing Placement information
// AND the validation is being done on a managed cluster
// THEN the validation should succeed.
func TestValidationSuccessForMultiClusterComponentCreationWithoutTargetClustersOnManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterComponentValidator()
	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.MCRegistrationSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	mcc := v1alpha12.MultiClusterComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mccomponent-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterComponentSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "invalid-cluster-name"}},
			},
		},
	}
	asrt.NoError(v.client.Create(context.TODO(), &s))

	req := newAdmissionRequest(admissionv1beta1.Create, mcc)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster component validation to succeed with missing placement information on managed cluster.")

	req = newAdmissionRequest(admissionv1beta1.Update, mcc)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster component validation to succeed with missing placement information on managed cluster.")
}
