// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherbackup

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	return scheme
}

// TestRancherBackupPostUninstall tests the post uninstall process for Rancher Backup
// GIVEN a call to postUninstall
// WHEN the objects exist in the cluster
// THEN no error is returned and all objects are deleted
func TestRancherBackupPostUninstall(t *testing.T) {
	assert := asserts.New(t)
	vz := v1alpha1.Verrazzano{}

	rancherCrb := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "rancher-backup",
		},
	}

	tests := []struct {
		name           string
		objects        []clipkg.Object
		nonRancherTest bool
	}{
		{
			name: "test rancher backup",
			objects: []clipkg.Object{
				&rancherCrb,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
			ctx := spi.NewFakeContext(c, &vz, nil, false)
			err := postUninstall(ctx)
			assert.NoError(err)

			// ClusterRoleBinding should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: "rancher-backup"}, &rbacv1.ClusterRoleBinding{})
			assert.NotNil(err)
			assert.True(errors.IsNotFound(err))
		})
	}
}
