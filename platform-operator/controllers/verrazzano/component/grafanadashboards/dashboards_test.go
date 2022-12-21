// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafanadashboards

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"testing"

	asserts "github.com/stretchr/testify/assert"
)

const (
	profileDir = "../../../../manifests/profiles"
)

func TestAppendOverrides(t *testing.T) {
	a := asserts.New(t)

	// GIVEN a call to append the Helm overrides
	//
	//	WHEN Istio is enabled
	//	THEN no overrides are created
	istioEnabledVZ := &vzapi.Verrazzano{}
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, nil, istioEnabledVZ, false, profileDir), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	a.Nil(err)
	a.Empty(kvs)

	// GIVEN a call to append the Helm overrides
	//
	//	WHEN Istio is disabled
	//	THEN an istioEnabled=false override is set
	falseVal := false
	istioDisabledVZ := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					Enabled: &falseVal,
				},
			},
		},
	}
	kvs, err = AppendOverrides(spi.NewFakeContext(nil, nil, istioDisabledVZ, false, profileDir), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	a.Nil(err)
	a.Equal(1, len(kvs))
	a.Equal("istioEnabled", kvs[0].Key)
	a.Equal("false", kvs[0].Value)
}
