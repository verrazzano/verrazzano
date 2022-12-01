// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"fmt"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakes "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestUpdateRancherClusterRoleBinding(t *testing.T) {
	a := asserts.New(t)

	vmcNoID := &v1alpha1.VerrazzanoManagedCluster{}

	clusterID := "testID"
	vmcID := vmcNoID.DeepCopy()
	vmcID.Status.RancherRegistration.ClusterID = clusterID

	clusterUserNoData := &unstructured.Unstructured{}
	clusterUserNoData.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   APIGroupRancherManagement,
		Version: APIGroupVersionRancherManagement,
		Kind:    UserKind,
	})
	clusterUserNoData.SetName(vzconst.VerrazzanoClusterRancherUsername)

	clusterUserData := clusterUserNoData.DeepCopy()
	data := clusterUserData.UnstructuredContent()
	data[UserUsernameAttribute] = vzconst.VerrazzanoClusterRancherUsername

	tests := []struct {
		name         string
		vmc          *v1alpha1.VerrazzanoManagedCluster
		expectCreate bool
		expectErr    bool
		user         *unstructured.Unstructured
	}{
		{
			name:         "test nil vmc",
			expectCreate: false,
			expectErr:    false,
			user:         clusterUserData,
		},
		{
			name:         "test vmc no cluster id",
			vmc:          vmcNoID,
			expectCreate: false,
			expectErr:    false,
			user:         clusterUserData,
		},
		{
			name:         "test vmc with cluster id",
			vmc:          vmcID,
			expectCreate: true,
			expectErr:    false,
			user:         clusterUserData,
		},
		{
			name:         "test user doesn't exist",
			vmc:          vmcID,
			expectCreate: false,
			expectErr:    true,
		},
		{
			name:         "test user no username",
			vmc:          vmcID,
			expectCreate: false,
			expectErr:    true,
			user:         clusterUserNoData,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := fakes.NewClientBuilder()
			if tt.user != nil {
				b = b.WithObjects(tt.user)
			}
			c := b.Build()

			r := &VerrazzanoManagedClusterReconciler{
				Client: c,
				log:    vzlog.DefaultLogger(),
			}
			err := r.updateRancherClusterRoleBindingTemplate(tt.vmc)

			if tt.expectErr {
				a.Error(err)
				return
			}
			a.NoError(err)

			if tt.expectCreate {
				name := fmt.Sprintf("crtb-verrazzano-cluster-%s", clusterID)
				resource := &unstructured.Unstructured{}
				resource.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   APIGroupRancherManagement,
					Version: APIGroupVersionRancherManagement,
					Kind:    ClusterRoleTemplateBindingKind,
				})
				err = c.Get(context.TODO(), types.NamespacedName{Namespace: clusterID, Name: name}, resource)
				a.NoError(err)
				a.NotNil(resource)
				a.Equal(clusterID, resource.Object[ClusterRoleTemplateBindingAttributeClusterName])
				a.Equal(vzconst.VerrazzanoClusterRancherUsername, resource.Object[ClusterRoleTemplateBindingAttributeUserName])
				a.Equal(vzconst.VerrazzanoClusterRancherName, resource.Object[ClusterRoleTemplateBindingAttributeRoleTemplateName])
			}
		})
	}
}
