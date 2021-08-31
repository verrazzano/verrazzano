// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package component

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"go.uber.org/zap"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// Test_appendWeblogicOperatorOverridesExtraKVs tests the appendWeblogicOperatorOverrides fn
// GIVEN a call to appendWeblogicOperatorOverrides
//  WHEN I call with no extra kvs
//  THEN the correct number of keyValue objects are returned and no errors occur
func Test_appendWeblogicOperatorOverrides(t *testing.T) {
	kvs, err := appendWeblogicOperatorOverrides(zap.S(), "weblogic-operator", "verrazzano-system", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 4)
}

// Test_appendWeblogicOperatorOverridesExtraKVs tests the appendWeblogicOperatorOverrides fn
// GIVEN a call to appendWeblogicOperatorOverrides
//  WHEN I pass in a keyValue list
//  THEN the values passed in are preserved and no errors occur
func Test_appendWeblogicOperatorOverridesExtraKVs(t *testing.T) {
	kvs := []bom.KeyValue{
		{Key: "key", Value: "value"},
	}
	var err error
	kvs, err = appendWeblogicOperatorOverrides(zap.S(), "weblogic-operator", "verrazzano-system", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 5)
}

// Test_weblogicOperatorPreInstall tests the weblogicOperatorPreInstall fn
// GIVEN a call to this fn
//  WHEN I call weblogicOperatorPreInstall
//  THEN no errors are returned
func Test_weblogicOperatorPreInstall(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	kvs, err := weblogicOperatorPreInstall(zap.S(), client, "weblogic-operator", "verrazzano-system", "")
	assert.NoError(t, err)
	assert.Len(t, kvs, 0)
}
