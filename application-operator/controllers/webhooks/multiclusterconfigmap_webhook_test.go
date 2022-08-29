// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	v1alpha12 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// newMultiClusterConfigmapValidator creates a new MultiClusterConfigmapValidator
func newMultiClusterConfigmapValidator() MultiClusterConfigmapValidator {
	scheme := newScheme()
	decoder, _ := admission.NewDecoder(scheme)
	cli := fake.NewClientBuilder().WithScheme(scheme).Build()
	v := MultiClusterConfigmapValidator{client: cli, decoder: decoder}
	return v
}

// TestValidationFailureForMultiClusterConfigMapCreationWithoutTargetClusters tests preventing the creation
// of a MultiClusterConfigMap resources that is missing Placement information.
// GIVEN a call to validate a MultiClusterConfigMap resource
// WHEN the MultiClusterConfigMap resource is missing Placement information
// THEN the validation should fail.
func TestValidationFailureForMultiClusterConfigMapCreationWithoutTargetClusters(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterConfigmapValidator()
	p := v1alpha12.MultiClusterConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcconfigmap-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterConfigMapSpec{},
	}

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster configmap validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "target cluster")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster configmap validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "target cluster")
}

// TestValidationFailureForMultiClusterConfigMapCreationTargetingMissingManagedCluster tests preventing the creation
// of a MultiClusterConfigMap resources that references a non-existent managed cluster.
// GIVEN a call to validate a MultiClusterConfigMap resource
// WHEN the MultiClusterConfigMap resource references a VerrazzanoManagedCluster that does not exist
// THEN the validation should fail.
func TestValidationFailureForMultiClusterConfigMapCreationTargetingMissingManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterConfigmapValidator()
	p := v1alpha12.MultiClusterConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcconfigmap-name",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha12.MultiClusterConfigMapSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "invalid-cluster-name"}},
			},
		},
	}

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster configmap validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "invalid-cluster-name")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.False(res.Allowed, "Expected multi-cluster configmap validation to fail due to missing placement information.")
	asrt.Contains(res.Result.Reason, "invalid-cluster-name")
}

// TestValidationSuccessForMultiClusterConfigMapCreationTargetingExistingManagedCluster tests allowing the creation
// of a MultiClusterConfigMap resources that references an existent managed cluster.
// GIVEN a call to validate a MultiClusterConfigMap resource
// WHEN the MultiClusterConfigMap resource references a VerrazzanoManagedCluster that does exist
// THEN the validation should pass.
func TestValidationSuccessForMultiClusterConfigMapCreationTargetingExistingManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterConfigmapValidator()
	c := v1alpha1.VerrazzanoManagedCluster{
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
	p := v1alpha12.MultiClusterConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcconfigmap-name",
			Namespace: "application-ns",
		},
		Spec: v1alpha12.MultiClusterConfigMapSpec{
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

	asrt.NoError(v.client.Create(context.TODO(), &c))
	asrt.NoError(v.client.Create(context.TODO(), &vp))

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster configmap create validation to succeed.")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster configmap update validation to succeed.")
}

// TestValidationSuccessForMultiClusterConfigMapCreationWithoutTargetClustersOnManagedCluster tests allowing the creation
// of a MultiClusterConfigMap resources that is missing target cluster information when on managed cluster.
// GIVEN a call to validate a MultiClusterConfigMap resource
// WHEN the MultiClusterConfigMap resource is missing Placement information
// AND the validation is being done on a managed cluster
// THEN the validation should succeed.
func TestValidationSuccessForMultiClusterConfigMapCreationWithoutTargetClustersOnManagedCluster(t *testing.T) {
	asrt := assert.New(t)
	v := newMultiClusterConfigmapValidator()
	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.MCRegistrationSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	p := v1alpha12.MultiClusterConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mcconfigmap-name",
			Namespace: "application-ns",
		},
		Spec: v1alpha12.MultiClusterConfigMapSpec{
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

	req := newAdmissionRequest(admissionv1.Create, p)
	res := v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster configmap validation to succeed with missing placement information on managed cluster.")

	req = newAdmissionRequest(admissionv1.Update, p)
	res = v.Handle(context.TODO(), req)
	asrt.True(res.Allowed, "Expected multi-cluster configmap validation to succeed with missing placement information on managed cluster.")
}

// TestMultiClusterConfigmapHandleFailed tests to make sure the failure metric is being exposed
// GIVEN a call to validate a MultiClusterConfigMap resource
// WHEN the MultiClusterConfigMap resource is failing
// THEN the validation should fail.
func TestMultiClusterConfigmapHandleFailed(t *testing.T) {
	assert := assert.New(t)
	mcc := v1alpha12.MultiClusterComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-mccomponent-error",
			Namespace: "application-ns",
		},
		Spec: v1alpha12.MultiClusterComponentSpec{
			Placement: v1alpha12.Placement{
				Clusters: []v1alpha12.Cluster{{Name: "valid-cluster-name"}},
			},
		},
	}
	// Create a request to Handle
	v := newMultiClusterComponentValidator()
	req := newAdmissionRequest(admissionv1.Create, mcc)
	v.Handle(context.TODO(), req)
	reconcileerrorCounterObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.MultiClusterConfigmapHandleError)
	assert.NoError(err)
	// Expect a call to fetch the error
	reconcileFailedCounterBefore := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	reconcileerrorCounterObject.Get().Inc()
	reconcileFailedCounterAfter := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)
}
