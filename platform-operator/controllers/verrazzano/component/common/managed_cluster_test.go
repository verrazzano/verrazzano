// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestManagedClusterRegistrationSecret tests fetching of managed cluster registration secret if it exists
func TestManagedClusterRegistrationSecret(t *testing.T) {
	a := assert.New(t)
	cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects().Build()
	a.Nil(GetManagedClusterRegistrationSecret(cli))
}
