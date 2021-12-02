// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewServiceAccount(t *testing.T) {
	namespace := "verrazzano"
	name := "test-serviceAccount"
	labels := map[string]string{"label1": "test", "label2": "test2"}
	imagePullSecret1 := "some-secret1"
	imagePullSecret2 := "some-secret2"

	serviceAccount := NewServiceAccount(namespace, name, []string{imagePullSecret1, imagePullSecret2}, labels)

	assert.Equalf(t, namespace, serviceAccount.Namespace, "Expected namespace did not match")
	assert.Equalf(t, name, serviceAccount.Name, "Expected service account name did not match")
	assert.Equalf(t, labels, serviceAccount.Labels, "Expected labels did not match")
	assert.Lenf(t, serviceAccount.ImagePullSecrets, 2, "Wrong number of image pull secrets")
	assert.Equalf(t, imagePullSecret1, serviceAccount.ImagePullSecrets[0].Name, "Wrong value of image pull secret")
	assert.Equalf(t, imagePullSecret2, serviceAccount.ImagePullSecrets[1].Name, "Wrong value of image pull secret")
}
