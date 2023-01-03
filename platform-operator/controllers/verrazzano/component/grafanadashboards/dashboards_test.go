// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafanadashboards

import (
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
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
	istioEnabledVZ := &v1alpha1.Verrazzano{}
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, istioEnabledVZ, nil, false, profileDir), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	a.Nil(err)
	a.Empty(kvs)

	// GIVEN a call to append the Helm overrides
	//
	//	WHEN Istio is disabled
	//	THEN an istioEnabled=false override is set
	falseVal := false
	istioDisabledVZ := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Istio: &v1alpha1.IstioComponent{
					Enabled: &falseVal,
				},
			},
		},
	}
	kvs, err = AppendOverrides(spi.NewFakeContext(nil, istioDisabledVZ, nil, false, profileDir), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	a.Nil(err)
	a.Equal(1, len(kvs))
	a.Equal("istioEnabled", kvs[0].Key)
	a.Equal("false", kvs[0].Value)
}
