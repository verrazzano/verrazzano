// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package contorllers

import (
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	configMapKind = "ConfigMap"
	secretKind    = "Secret"
)

// vzContainsResource checks to see if the resource is listed in the Verrazzano
func vzContainsResource(ctx spi.ComponentContext, object client.Object) bool {
	for _, component := range registry.GetComponents() {
		if found := componentContainsResource(component.GetHelmOverrides(ctx), object); found {
			return found
		}
	}
	return false
}

// componentContainsResource looks through the component override list see if the resource is listed
func componentContainsResource(Overrides []installv1alpha1.Overrides, object client.Object) bool {
	objectKind := object.GetObjectKind().GroupVersionKind().Kind
	for _, override := range Overrides {
		if objectKind == configMapKind && override.ConfigMapRef != nil {
			if object.GetName() == override.ConfigMapRef.Name {
				return true
			}
		}
		if objectKind == secretKind && override.SecretRef != nil {
			if object.GetName() == override.SecretRef.Name {
				return true
			}
		}
	}
	return false
}
