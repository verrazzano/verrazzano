// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package networkpolicies

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// GIVEN a ComponentContext and empty KeyValue array
//
//	WHEN the appendOverrides function is called
//	THEN we expect a single KeyValue item is added to the KeyValue array
func TestAppendOverrides(t *testing.T) {
	ctx := spi.NewFakeContext(nil, &vzapi.Verrazzano{}, nil, false)
	kvs := []bom.KeyValue{}
	kvs, err := appendOverrides(ctx, "", "", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
}
