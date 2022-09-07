// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	v1alpha12 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// newMultiClusterApplicationConfigurationValidator creates a new MultiClusterApplicationConfigurationValidator
func newMultiClusterApplicationConfigurationValidator() MultiClusterApplicationConfigurationValidator {
	scheme := newScheme()
	decoder, _ := admission.NewDecoder(scheme)
	cli := fake.NewClientBuilder().WithScheme(scheme).Build()
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
	p := v1alpha12.MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterApplicationConfigurationSpec{},
	}

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster application configuration validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "target cluster")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster application configuration validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "target cluster")
}

// TestValidationFailureForMultiClusterApplicationConfigurationCreationTargetingMissingManagedCluster tests preventing the creation
// of a MultiClusterApplicationConfiguration resources that references a non-existent managed cluster.
// GIVEN a call to validate a MultiClusterApplicationConfiguration resource
// WHEN the MultiClusterApplicationConfiguration resource references a VerrazzanoManagedCluster that does not exist
// THEN the validation should fail.
func TestValidationFailureForMultiClusterApplicationConfigurationCreationTargetingMissingManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterApplicationConfigurationValidator()
	p := v1alpha12.MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterApplicationConfigurationSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "invalid-cluster-name"}},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster application configuration validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "invalid-cluster-name")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster application configuration validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "invalid-cluster-name")
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
			CASecret:                     "test-secret",
			ManagedClusterManifestSecret: "test-cluster-manifest-secret",
			ServiceAccount:               "test-service-account",
		},
	}
	mcac := v1alpha12.MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: "application-ns",
		},
		Spec: v1alpha12.MultiClusterApplicationConfigurationSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "valid-cluster-name"}},
			},
		},
	}
	vp := v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-verrazzanoproject-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "application-ns",
						},
					},
				},
			},
		},
	}

	asrt.NoError(v.client.Create(context.TODO(), &mc))
	asrt.NoError(v.client.Create(context.TODO(), &vp))

	req := newAdmissionRequest(admissionv1.Create, mcac)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster application configuration create validation to succeed.")

	req = newAdmissionRequest(admissionv1.Update, mcac)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster application configuration update validation to succeed.")
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
			Name:      constants.MCRegistrationSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	mcac := v1alpha12.MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: "application-ns",
		},
		Spec: v1alpha12.MultiClusterApplicationConfigurationSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "invalid-cluster-name"}},
			},
		},
	}
	vp := v1alpha12.VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-verrazzanoproject-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.VerrazzanoProjectSpec{
			Template: v1alpha12.ProjectTemplate{
				Namespaces: []v1alpha12.NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "application-ns",
						},
					},
				},
			},
		},
	}

	asrt.NoError(v.client.Create(context.TODO(), &s))
	asrt.NoError(v.client.Create(context.TODO(), &vp))

	req := newAdmissionRequest(admissionv1.Create, mcac)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster application configuration validation to succeed with missing placement information on managed cluster.")

	req = newAdmissionRequest(admissionv1.Update, mcac)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster application configuration validation to succeed with missing placement information on managed cluster.")
}

// TestValidateSecrets tests the function validateSecrets
// GIVEN a call to validateSecrets
// WHEN called with various MultiClusterApplicationConfiguration resources
// THEN the validation should succeed or fail based on what secrets are specified in the
//   MultiClusterApplicationConfiguration resource
func TestValidateSecrets(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterApplicationConfigurationValidator()

	mcac := &v1alpha12.MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterApplicationConfigurationSpec{},
	}

	// No secrets specified, so success is expected
	asrt.NoError(v.validateSecrets(mcac))

	mcac = &v1alpha12.MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterApplicationConfigurationSpec{
			Secrets: []string{
				"secret1",
			},
		},
	}

	// Secret not found, so failure expected
	err := v.validateSecrets(mcac)
	asrt.EqualError(err, "secret(s) secret1 specified in MultiClusterApplicationConfiguration not found in namespace verrazzano-mc")

	mcac = &v1alpha12.MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterApplicationConfigurationSpec{
			Secrets: []string{
				"secret1",
				"secret2",
			},
		},
	}

	// Secrets not found, so failure expected
	err = v.validateSecrets(mcac)
	asrt.EqualError(err, "secret(s) secret1,secret2 specified in MultiClusterApplicationConfiguration not found in namespace verrazzano-mc")

	mcac = &v1alpha12.MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcapplicationconfiguration-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterApplicationConfigurationSpec{
			Secrets: []string{
				"secret1",
			},
		},
	}
	secret1 := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	asrt.NoError(v.client.Create(context.TODO(), &secret1))

	// Secret should be found, so success is expected
	asrt.NoError(v.validateSecrets(mcac))

}

// TestMultiClusterAppConfigHandleFailed tests to make sure the failure metric is being exposed
func TestMultiClusterAppConfigHandleFailed(t *testing.T) {
	assert := assert.New(t)
	// Create a request and decode(Handle)
	decoder := decoder()
	defaulter := &IstioWebhook{}
	_ = defaulter.InjectDecoder(decoder)
	req := admission.Request{}
	defaulter.Handle(context.TODO(), req)
	reconcileerrorCounterObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.MultiClusterAppconfigPodHandleError)
	assert.NoError(err)
	// Expect a call to fetch the error
	reconcileFailedCounterBefore := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	reconcileerrorCounterObject.Get().Inc()
	reconcileFailedCounterAfter := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)
}
