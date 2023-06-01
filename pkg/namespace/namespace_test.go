// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	corev1Cli "k8s.io/client-go/kubernetes/typed/core/v1"
	"testing"
)

// TestCheckIfVerrazzanoManagedNamespaceExists tests the CheckIfVerrazzanoManagedNamespaceExists fn
// GIVEN a call to CheckIfVerrazzanoManagedNamespaceExists
// WHEN the ns exists and has the Verrzzano namespace label
// THEN true is retured without error
func TestCheckIfVerrazzanoManagedNamespaceExists(t *testing.T) {
	const namespace = "somens"
	k8sutil.GetCoreV1Func = func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						constants.VerrazzanoManagedKey: namespace,
					},
				},
			},
		).CoreV1(), nil
	}
	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()

	exists, err := CheckIfVerrazzanoManagedNamespaceExists(namespace)
	assert.NoError(t, err)
	assert.True(t, exists)
}

// TestCheckIfUnmanagedNamespaceExists tests the CheckIfVerrazzanoManagedNamespaceExists fn
// GIVEN a call to CheckIfVerrazzanoManagedNamespaceExists
// WHEN the ns exists and does NOT have the Verrzzano namespace label
// THEN false is retured without error
func TestCheckIfUnmanagedNamespaceExists(t *testing.T) {
	const namespace = "somens"
	k8sutil.GetCoreV1Func = func(_ ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			},
		).CoreV1(), nil
	}
	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()

	exists, err := CheckIfVerrazzanoManagedNamespaceExists(namespace)
	assert.NoError(t, err)
	assert.False(t, exists)
}
