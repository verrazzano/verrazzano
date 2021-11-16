// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package mysql

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	"testing"
)

func TestCreateDBFile(t *testing.T) {
	vz := &vzapi.Verrazzano{}
	err := createDBFile(spi.NewContext(zap.S(), nil, vz, false))
	assert.Nil(t, err, "error creating db file")
}
