// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

// newMultiClusterApplicationConfigurationValidator creates a new MultiClusterApplicationConfigurationValidator
func newMultiClusterApplicationConfigurationValidator() MultiClusterApplicationConfigurationValidator {
	scheme := newScheme()
	decoder, _ := admission.NewDecoder(scheme)
	cli := fake.NewFakeClientWithScheme(scheme)
	v := MultiClusterApplicationConfigurationValidator{client: cli, decoder: decoder}
	return v
}

// TestValidationFailureForMultiClusterApplicationConfigurationCreationWithoutTargetClusters tests preventing the creation
// of a MultiClusterApplicationConfiguration resources that is missing Placement information.
// GIVEN a call to validate a MultiClusterApplicationConfiguration resource
// WHEN the MultiClusterApplicationConfiguration resource is missing Placement information
// THEN the validation should fail.
func TestValidationFailureForMultiClusterApplicationConfigurationCreationWithoutTargetClusters(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterApplicationConfigurationValidator()
	p := MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: MultiClusterApplicationConfigurationSpec{},
	}
	req := newAdmissionRequest(admissionv1beta1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster application configuration validation to fail due to missing placement information.")
}

// TestValidationFailureForMultiClusterApplicationConfigurationCreationTargetingMissingManagedCluster tests preventing the creation
// of a MultiClusterApplicationConfiguration resources that references a non-existent managed cluster.
// GIVEN a call to validate a MultiClusterApplicationConfiguration resource
// WHEN the MultiClusterApplicationConfiguration resource references a VerrazzanoManagedCluster that does not exist
// THEN the validation should fail.
func TestValidationFailureForMultiClusterApplicationConfigurationCreationTargetingMissingManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterApplicationConfigurationValidator()
	p := MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: MultiClusterApplicationConfigurationSpec{
			Placement: Placement{
				Clusters: []Cluster{{Name: "invalid-cluster-name"}},
			},
		},
	}
	req := newAdmissionRequest(admissionv1beta1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster application configuration validation to fail due to missing placement information.")
}

// TestValidationSuccessForMultiClusterApplicationConfigurationCreationTargetingExistingManagedCluster tests allowing the creation
// of a MultiClusterApplicationConfiguration resources that references an existent managed cluster.
// GIVEN a call to validate a MultiClusterApplicationConfiguration resource
// WHEN the MultiClusterApplicationConfiguration resource references a VerrazzanoManagedCluster that does exist
// THEN the validation should pass.
func TestValidationSuccessForMultiClusterApplicationConfigurationCreationTargetingExistingManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterApplicationConfigurationValidator()
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
	mcc := MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: MultiClusterApplicationConfigurationSpec{
			Placement: Placement{
				Clusters: []Cluster{{Name: "valid-cluster-name"}},
			},
		},
	}
	asrt.NoError(v.client.Create(context.TODO(), &mc))
	req := newAdmissionRequest(admissionv1beta1.Create, mcc)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster application configuration create validation to succeed.")
}

// TestValidationSuccessForMultiClusterApplicationConfigurationCreationWithoutTargetClustersOnManagedCluster tests allowing the creation
// of a MultiClusterApplicationConfiguration resources that is missing target cluster information when on managed cluster.
// GIVEN a call to validate a MultiClusterApplicationConfiguration resource
// WHEN the MultiClusterApplicationConfiguration resource is missing Placement information
// AND the validation is being done on a managed cluster
// THEN the validation should succeed.
func TestValidationSuccessForMultiClusterApplicationConfigurationCreationWithoutTargetClustersOnManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterApplicationConfigurationValidator()
	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "verrazzano-cluster-registration",
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	mcc := MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: MultiClusterApplicationConfigurationSpec{
			Placement: Placement{
				Clusters: []Cluster{{Name: "invalid-cluster-name"}},
			},
		},
	}
	asrt.NoError(v.client.Create(context.TODO(), &s))
	req := newAdmissionRequest(admissionv1beta1.Create, mcc)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster application configuration validation to succeed with missing placement information on managed cluster.")
}
