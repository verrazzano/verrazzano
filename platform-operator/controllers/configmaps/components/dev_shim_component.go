// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package components

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
)

func newDevShimComponent(cm *corev1.ConfigMap) (spi.Component, error) {
	componentName, ok := cm.Data[componentNameKey]
	if !ok {
		return nil, fmt.Errorf("ConfigMap %s does not contain the name field, cannot reconcile component", cm.Name)
	}

	shimComponent, found := shimComponents[componentName]
	if !found {
		return nil, fmt.Errorf("Component name %s in configMap %s is not a valid shimComponent", componentName, cm.Name)
	}

	return shimComponent, nil
}
