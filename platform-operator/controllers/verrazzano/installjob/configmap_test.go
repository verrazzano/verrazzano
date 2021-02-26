// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigMap(t *testing.T) {
	name, namespace := "test-configmap", "verrazzano"
	labels := map[string]string{"label1": "test", "label2": "test2"}

	configMap := NewConfigMap(namespace, name, labels)
	assert.Equal(t, name, configMap.Name, "Expected name did not match")
	assert.Equalf(t, namespace, configMap.Namespace, "Expected namespace did not match")
	assert.Equalf(t, labels, configMap.Labels, "Expected labels did not match")
}
