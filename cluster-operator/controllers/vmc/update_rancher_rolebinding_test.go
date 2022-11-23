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

	tests := []struct {
		name         string
		vmc          *v1alpha1.VerrazzanoManagedCluster
		expectCreate bool
	}{
		{
			name:         "test nil vmc",
			expectCreate: false,
		},
		{
			name:         "test vmc no cluster id",
			vmc:          vmcNoID,
			expectCreate: false,
		},
		{
			name:         "test vmc with cluster id",
			vmc:          vmcID,
			expectCreate: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fakes.NewClientBuilder().Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: c,
				log:    vzlog.DefaultLogger(),
			}
			err := r.UpdateRancherClusterRoleBindingTemplate(tt.vmc)
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
				a.Equal(vzconst.VerrazzanoClusterRancherUser, resource.Object[ClusterRoleTemplateBindingAttributeUserName])
				a.Equal(vzconst.VerrazzanoClusterRancherRole, resource.Object[ClusterRoleTemplateBindingAttributeRoleTemplateName])
			}
		})
	}
}
