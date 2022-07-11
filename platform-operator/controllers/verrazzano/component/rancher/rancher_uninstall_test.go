// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	admv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestPostUninstall tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the objects exist in the cluster
// THEN no error is returned and all objects are deleted
func TestPostUninstall(t *testing.T) {
	assert := asserts.New(t)
	vz := v1alpha1.Verrazzano{}

	nonRanNSName := "not-rancher"
	rancherNSName := "cattle-system"
	rancherNSName2 := "fleet-rancher"
	rancherCrName := "fleet-system"
	randCR := "randomCR"
	randCRB := "randomCRB"

	nonRancherNs := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nonRanNSName,
		},
	}
	rancherNs := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: rancherNSName,
		},
	}
	rancherNs2 := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: rancherNSName2,
		},
	}
	mutWebhook := admv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
	}
	valWebhook := admv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
	}
	crRancher := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: rancherCrName,
		},
	}
	crbRancher := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
	}
	crNotRancher := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: randCR,
		},
	}
	crbNotRancher := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: randCRB,
		},
	}
	controllerCM := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: controllerCMName,
		},
	}
	lockCM := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: lockCMName,
		},
	}

	tests := []struct {
		name           string
		objects        []clipkg.Object
		nonRancherTest bool
	}{
		{
			name: "test empty cluster",
		},
		{
			name: "test non Rancher ns",
			objects: []clipkg.Object{
				&nonRancherNs,
			},
		},
		{
			name: "test Rancher ns",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
			},
		},
		{
			name: "test multiple Rancher ns",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
			},
		},
		{
			name: "test mutating webhook",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
			},
		},
		{
			name: "test validating webhook",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
			},
		},
		{
			name: "test CR and CRB",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
				&crRancher,
				&crbRancher,
			},
		},
		{
			name: "test non Rancher CR and CRB",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
				&crRancher,
				&crbRancher,
				&crNotRancher,
				&crbNotRancher,
			},
			nonRancherTest: true,
		},
		{
			name: "test config maps",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
				&crRancher,
				&crbRancher,
				&crNotRancher,
				&crbNotRancher,
				&controllerCM,
				&lockCM,
			},
		},
	}
	setRancherSystemTool("echo")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
			ctx := spi.NewFakeContext(c, &vz, false)
			err := postUninstall(ctx)
			assert.NoError(err)

			// MutatingWebhookConfiguration should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &admv1.MutatingWebhookConfiguration{})
			assert.True(apierrors.IsNotFound(err))
			// ValidatingWebhookConfiguration should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &admv1.ValidatingWebhookConfiguration{})
			assert.True(apierrors.IsNotFound(err))
			// ClusterRole should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: rancherCrName}, &rbacv1.ClusterRole{})
			assert.True(apierrors.IsNotFound(err))
			// ClusterRoleBinding should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &rbacv1.ClusterRoleBinding{})
			assert.True(apierrors.IsNotFound(err))
			if tt.nonRancherTest {
				// Verify that non-Rancher components did not get cleaned up
				err = c.Get(context.TODO(), types.NamespacedName{Name: randCR}, &rbacv1.ClusterRole{})
				assert.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: randCRB}, &rbacv1.ClusterRoleBinding{})
				assert.Nil(err)
			}
			// ConfigMaps should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: controllerCMName}, &v1.ConfigMap{})
			assert.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: lockCMName}, &v1.ConfigMap{})
			assert.True(apierrors.IsNotFound(err))
		})
	}
}
