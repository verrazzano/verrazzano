// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafanadashboards

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// AppendOverrides builds the set of Grafana dashboard overrides for the Helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	if !vzcr.IsIstioEnabled(compContext.EffectiveCR()) {
		kvs = append(kvs, bom.KeyValue{Key: "istioEnabled", Value: "false"})
	}
	return kvs, nil
}
