// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestNewClusterRoleBinding tests the creation of a clusterRoleBinding
// GIVEN a verrazzano.install.verrazzano.io custom resource, clusterRoleBinding name, and serviceAccount name
//
//	WHEN I call NewClusterRoleBinding
//	THEN a cluster role binding is created and verified
func TestNewClusterRoleBinding(t *testing.T) {
	vz := installv1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-install",
			Namespace: "verrazzano",
			Labels:    map[string]string{"label1": "test", "label2": "test2"},
		},
	}

	namespace := "verrazzano"
	name := "test-clusterRoleBinding"
	saName := "service-account"

	clusterRoleBinding := NewClusterRoleBinding(&vz, name, namespace, saName)

	assert.Equalf(t, "", clusterRoleBinding.Namespace, "Expected namespace did not match")
	assert.Equalf(t, name, clusterRoleBinding.Name, "Expected clusterRoleBinding name did not match")
	assert.Equalf(t, 0, len(clusterRoleBinding.OwnerReferences), "Expected length of owner references did not match")
	assert.Equalf(t, vz.Labels, clusterRoleBinding.Labels, "Expected labels did not match")
	assert.Equalf(t, saName, clusterRoleBinding.Subjects[0].Name, "Expected service account name did not match")
	assert.Equalf(t, "ServiceAccount", clusterRoleBinding.Subjects[0].Kind, "Expected subject kind did not match")
	assert.Equalf(t, namespace, clusterRoleBinding.Subjects[0].Namespace, "Expected namespace did not match")
}
