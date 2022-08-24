// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"testing"
	"time"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	admv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

	nonRanNSName := "local-not-rancher"
	rancherNSName := "local"
	rancherNSName2 := "fleet-rancher"
	rancherCrName := "proxy-1234"
	mwcName := "mutating-webhook-configuration"
	vwcName := "validating-webhook-configuration"
	pvName := "pvc-12345"
	pv2Name := "ocid1.volume.oc1.ca-toronto-1.12345"
	rbName := "rb-test"
	nonRancherRBName := "testrb"
	randPV := "randomPV"
	randCR := "randomCR"
	randCRB := "randomCRB"
	rancherCRDName := "definitelyrancher.cattle.io"
	nonRancherCRDName := "other.cattle"

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
	mutWebhook2 := admv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: mwcName,
		},
	}
	valWebhook := admv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
	}
	valWebhook2 := admv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: vwcName,
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
	rbRancher := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: rbName,
		},
	}
	rbNotRancher := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: nonRancherRBName,
		},
	}
	controllerCM := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllerCMName,
			Namespace: constants.KubeSystem,
		},
	}
	lockCM := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lockCMName,
			Namespace: constants.KubeSystem,
		},
	}
	rancherPV := v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}
	rancherPV2 := v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pv2Name,
		},
	}
	nonRancherPV := v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: randPV,
		},
	}
	delTimestamp := metav1.NewTime(time.Now())
	crdAPIVersion := "apiextensions.k8s.io/v1"
	crdKind := "CustomResourceDefinition"
	rancherCRD := v12.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       crdKind,
			APIVersion: crdAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              rancherCRDName,
			Finalizers:        []string{"somefinalizer"},
			DeletionTimestamp: &delTimestamp,
		},
	}
	nonRancherCRD := v12.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       crdKind,
			APIVersion: crdAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nonRancherCRDName,
		},
		Spec: v12.CustomResourceDefinitionSpec{
			Group:                 "cattle.io",
			Names:                 v12.CustomResourceDefinitionNames{},
			Scope:                 "",
			Versions:              nil,
			Conversion:            nil,
			PreserveUnknownFields: false,
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
				&mutWebhook2,
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
				&valWebhook2,
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
			name: "test non Rancher components",
			objects: []clipkg.Object{
				&nonRancherNs,
				&crNotRancher,
				&crbNotRancher,
				&nonRancherPV,
				&rbNotRancher,
				&nonRancherCRD,
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
		{
			name: "test persistent volume",
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
				&rancherPV,
				&rancherPV2,
			},
		},
		{
			name: "test clusterRole binding",
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
				&rancherPV,
				&rancherPV2,
				&rbRancher,
			},
		},
		{
			name: "test CRD finalizer cleanup",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
				&rancherCRD,
			},
		},
	}
	setRancherSystemTool("echo")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
			ctx := spi.NewFakeContext(c, &vz, false)

			crd1 := v12.CustomResourceDefinition{}
			c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd1)

			err := postUninstall(ctx)
			assert.NoError(err)

			// MutatingWebhookConfigurations should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &admv1.MutatingWebhookConfiguration{})
			assert.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: mwcName}, &admv1.MutatingWebhookConfiguration{})
			assert.True(apierrors.IsNotFound(err))
			// ValidatingWebhookConfigurations should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &admv1.ValidatingWebhookConfiguration{})
			assert.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: vwcName}, &admv1.ValidatingWebhookConfiguration{})
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
				err = c.Get(context.TODO(), types.NamespacedName{Name: randPV}, &v1.PersistentVolume{})
				assert.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: nonRancherRBName}, &rbacv1.RoleBinding{})
				assert.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: nonRancherCRDName}, &v12.CustomResourceDefinition{})
				assert.Nil(err)
			}
			// ConfigMaps should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: controllerCMName}, &v1.ConfigMap{})
			assert.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: lockCMName}, &v1.ConfigMap{})
			assert.True(apierrors.IsNotFound(err))
			// Persistent volume should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: pvName}, &v1.PersistentVolume{})
			assert.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: pv2Name}, &v1.PersistentVolume{})
			assert.True(apierrors.IsNotFound(err))
			// Role Binding should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: rbName}, &rbacv1.RoleBinding{})
			assert.True(apierrors.IsNotFound(err))
			// Rancher CRD finalizer should have been deleted which should cause it to go away
			// since it had a deletion timestamp
			crd := v12.CustomResourceDefinition{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd)
			assert.True(apierrors.IsNotFound(err))
		})
	}
}

// TestIsRancherNamespace tests the namespace belongs to Rancher
// GIVEN a call to isRancherNamespace
// WHEN the namespace belings to Rancher or not
// THEN we see true if it is and false if not
func TestIsRancherNamespace(t *testing.T) {
	assert := asserts.New(t)

	assert.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cattle-system",
		},
	}))
	assert.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
			Annotations: map[string]string{
				rancherSysNS: "true",
			},
		},
	}))
	assert.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
			Annotations: map[string]string{
				rancherSysNS: "false",
			},
		},
	}))
	assert.False(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
			Annotations: map[string]string{
				rancherSysNS: "false",
			},
		},
	}))
	assert.False(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
		},
	}))
}
