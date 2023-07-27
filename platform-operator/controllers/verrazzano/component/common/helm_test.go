// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var objWithExistingHelmOwner = v1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "some-ns",
		Name:      "some-name",
		Annotations: map[string]string{
			helmReleaseNameKey:      "some-release",
			helmReleaseNamespaceKey: "some-ns",
		},
		Labels: map[string]string{
			managedByLabelKey: "Helm",
		},
	},
}

var objNoHelmOwner = v1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "some-ns",
		Name:      "some-name",
		Annotations: map[string]string{
			"some/annot": "someval",
		},
		Labels: map[string]string{
			"somelabel": "somelabelval",
		},
	},
}

func TestAssociateHelmObject(t *testing.T) {
	tests := []struct {
		name string
		obj  *v1.Deployment
		keep bool
	}{
		{"Existing Helm owner, keep=true", &objWithExistingHelmOwner, true},
		{"Existing Helm owner, keep=false", &objWithExistingHelmOwner, false},
		{"No Helm owner, keep=true", &objNoHelmOwner, true},
		{"No Helm owner, keep=false", &objNoHelmOwner, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// deep copy the object since AssociateHelmObject makes inline changes
			myDeployment := v1.Deployment{}
			tt.obj.DeepCopyInto(&myDeployment)
			origLabels := myDeployment.Labels
			origAnnotations := myDeployment.Annotations

			fakeClient := fake.NewClientBuilder().WithObjects(&myDeployment).Build()
			newHelmReleaseName := types.NamespacedName{Namespace: "new-rel-namespace", Name: "new-rel-name"}
			objName := types.NamespacedName{Namespace: myDeployment.GetNamespace(), Name: myDeployment.GetName()}
			_, err := AssociateHelmObject(fakeClient, &myDeployment, newHelmReleaseName, objName, tt.keep)
			assert.NoError(t, err)

			newDeployment := v1.Deployment{}
			err = fakeClient.Get(context.TODO(), objName, &newDeployment)
			assert.NoError(t, err)
			assert.Equal(t, newHelmReleaseName.Name, newDeployment.Annotations[helmReleaseNameKey])
			assert.Equal(t, newHelmReleaseName.Namespace, newDeployment.Annotations[helmReleaseNamespaceKey])
			if tt.keep {
				assert.Equal(t, "keep", newDeployment.Annotations[helmResourcePolicyKey])
			} else {
				assert.Equal(t, "", newDeployment.Annotations[helmResourcePolicyKey])
			}
			assertOtherLabelsAnnotationsUnchanged(t, newDeployment, origAnnotations, origLabels)
		})
	}
}

func TestDisassociateHelmObject(t *testing.T) {
	tests := []struct {
		name          string
		obj           *v1.Deployment
		keep          bool
		managedByHelm bool
	}{
		{"Existing Helm owner, keep=true, managedByHelm", &objWithExistingHelmOwner, true, true},
		{"Existing Helm owner, keep=false, managedByHelm", &objWithExistingHelmOwner, false, true},
		{"No Helm owner, keep=true, NOT managed by Helm", &objNoHelmOwner, true, false},
		{"No Helm owner, keep=false, NOT managed by Helm", &objNoHelmOwner, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// deep copy the object since DisassociateHelmObject makes inline changes
			myDeployment := v1.Deployment{}
			tt.obj.DeepCopyInto(&myDeployment)
			origLabels := myDeployment.Labels
			origAnnotations := myDeployment.Annotations

			fakeClient := fake.NewClientBuilder().WithObjects(&myDeployment).Build()
			objName := types.NamespacedName{Namespace: myDeployment.GetNamespace(), Name: myDeployment.GetName()}
			_, err := DisassociateHelmObject(fakeClient, &myDeployment, objName, tt.keep, tt.managedByHelm)
			assert.NoError(t, err)

			newDeployment := v1.Deployment{}
			err = fakeClient.Get(context.TODO(), objName, &newDeployment)
			assert.NoError(t, err)

			assert.Equal(t, "", newDeployment.Annotations[helmReleaseNameKey])
			assert.Equal(t, "", newDeployment.Annotations[helmReleaseNamespaceKey])
			if tt.keep {
				assert.Equal(t, "keep", newDeployment.Annotations[helmResourcePolicyKey])
			} else {
				assert.Equal(t, "", newDeployment.Annotations[helmResourcePolicyKey])
			}
			if tt.managedByHelm {
				assert.Equal(t, "Helm", newDeployment.Labels[managedByLabelKey])
			} else {
				assert.Equal(t, "", newDeployment.Labels[managedByLabelKey])
			}
			assertOtherLabelsAnnotationsUnchanged(t, newDeployment, origAnnotations, origLabels)
		})
	}
}

func TestRemoveResourcePolicyAnnotation(t *testing.T) {
	withResourcePolicyAnnotation := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "some-ns",
			Name:      "some-name",
			Annotations: map[string]string{
				helmResourcePolicyKey: "Keep",
			},
		},
	}
	noResourcePolicyAnnotation := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "some-ns",
			Name:      "some-name",
			Annotations: map[string]string{
				helmResourcePolicyKey: "Keep",
			},
		},
	}

	tests := []struct {
		name string
		obj  *v1.Deployment
	}{
		{"with existing resource policy", &withResourcePolicyAnnotation},
		{"with existing resource policy", &noResourcePolicyAnnotation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// deep copy the object since DisassociateHelmObject makes inline changes
			myDeployment := v1.Deployment{}
			tt.obj.DeepCopyInto(&myDeployment)

			fakeClient := fake.NewClientBuilder().WithObjects(&myDeployment).Build()
			objName := types.NamespacedName{Namespace: myDeployment.GetNamespace(), Name: myDeployment.GetName()}
			_, err := RemoveResourcePolicyAnnotation(fakeClient, &myDeployment, objName)
			assert.NoError(t, err)

			newDeployment := v1.Deployment{}
			err = fakeClient.Get(context.TODO(), objName, &newDeployment)
			assert.NoError(t, err)
			assert.Equal(t, "", newDeployment.Annotations[helmResourcePolicyKey])
		})
	}
}

// assertOtherLabelsAnnotationsUnchanged asserts that labels and annotations except the ones used by Helm
// are retained unchanged
func assertOtherLabelsAnnotationsUnchanged(t *testing.T, newDeployment v1.Deployment, origAnnotations map[string]string, origLabels map[string]string) {
	// other annotations and labels are retained unchanged
	for key, val := range origAnnotations {
		if key == helmReleaseNamespaceKey || key == helmReleaseNameKey {
			continue
		}
		assert.Equal(t, val, newDeployment.Annotations[key])
	}
	for key, val := range origLabels {
		if key == managedByLabelKey {
			continue
		}
		assert.Equal(t, val, newDeployment.Labels[key])
	}
}
