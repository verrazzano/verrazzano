// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

func ApplyCRDYaml(ctx spi.ComponentContext, helmChartsDir string) error {
	path := filepath.Join(helmChartsDir, "/crds")
	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), "")
	ctx.Log().Oncef("Applying yaml for crds in %s", path)
	return yamlApplier.ApplyD(path)
}

// ConvertVerrazzanoCR converts older version of Verrzzano CR in v1alpha1.Verrazzano to newer version of v1beta1.Verrazzano
func ConvertVerrazzanoCR(vz *vzapi.Verrazzano, vzv1beta1 *v1beta1.Verrazzano) error {
	if vz == nil {
		return fmt.Errorf("Old VZ CR that needs to be upgraded, cannot be nil")
	}
	if err := vz.ConvertTo(vzv1beta1); err != nil {
		return err
	}
	return nil
}
