// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = vzapi.AddToScheme(testScheme)
	_ = clientgoscheme.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// TestCopySecret tests the CopySecret function
func TestCopySecret(t *testing.T) {
	secretName := "test-secret"
	invalidSecret := "nonExistentSecret"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("test"),
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(secret).Build()
	ctx := spi.NewFakeContext(fakeClient, nil, nil, false)
	err := CopySecret(ctx, secretName, constants.VerrazzanoSystemNamespace, "test secret")
	assert.NoError(t, err)
	destSecret := &corev1.Secret{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Name: secretName,
		Namespace: constants.VerrazzanoSystemNamespace}, destSecret)
	assert.NoError(t, err)
	assert.Equal(t, secret.Data, destSecret.Data)
	err = CopySecret(ctx, invalidSecret, constants.VerrazzanoSystemNamespace, "test secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret "+invalidSecret+" not found in namespace "+constants.VerrazzanoInstallNamespace)
}
