// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/stretchr/testify/assert"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"testing"
)

func TestIsQuickCreate(t *testing.T) {
	o := &Values{}
	assert.True(t, o.IsQuickCreate())
	o.Network = &vmcv1alpha1.Network{
		CreateVCN: false,
	}
	assert.False(t, o.IsQuickCreate())
	o.Network.CreateVCN = true
	assert.True(t, o.IsQuickCreate())
}
