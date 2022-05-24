// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// GetInstallOverridesYAML takes the list of Overrides and returns a string array of YAMLs
func GetInstallOverridesYAML(ctx spi.ComponentContext, overrides []v1alpha1.Overrides) ([]string, error) {
	var overrideStrings []string
	for _, override := range overrides {
		// Check if ConfigMapRef is populated and gather data
		if override.ConfigMapRef != nil {
			// Get the ConfigMap data
			yaml, err := getConfigMapOverrides(ctx, override.ConfigMapRef)
			if err != nil {
				return overrideStrings, err
			}
			overrideStrings = append(overrideStrings, yaml)
			continue
		}
		// Check if SecretRef is populated and gather data
		if override.SecretRef != nil {
			// Get the Secret data
			yaml, err := getSecretOverrides(ctx, override.SecretRef)
			if err != nil {
				return overrideStrings, err
			}
			overrideStrings = append(overrideStrings, yaml)
		}
	}
	return overrideStrings, nil
}

// getConfigMapOverrides takes a ConfigMap selector and returns the YAML data and handles k8s api errors appropriately
func getConfigMapOverrides(ctx spi.ComponentContext, selector *v1.ConfigMapKeySelector) (string, error) {
	configMap := &v1.ConfigMap{}
	nsn := types.NamespacedName{Name: selector.Name, Namespace: ctx.EffectiveCR().Namespace}
	optional := selector.Optional
	err := ctx.Client().Get(context.TODO(), nsn, configMap)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			ctx.Log().Errorf("Error retrieving ConfigMap %s: %v", nsn.Name, err)
			return "", err
		}
		if optional == nil || !*optional {
			err = ctx.Log().ErrorfThrottledNewErr("Could not get Configmap %s from namespace %s: %v", nsn.Name, nsn.Namespace, err)
			return "", err
		}
		ctx.Log().Infof("Optional Configmap %s from namespace %s not found", nsn.Name, nsn.Namespace)
		return "", nil
	}

	// Get resource data
	fieldData, ok := configMap.Data[selector.Key]
	if !ok {
		if optional == nil || !*optional {
			err := ctx.Log().ErrorfThrottledNewErr("Could not get Data field %s from Resource %s from namespace %s", selector.Key, nsn.Name, nsn.Namespace)
			return "", err
		}
		ctx.Log().Infof("Optional Resource %s from namespace %s missing Data key %s", nsn.Name, nsn.Namespace, selector.Key)
	}
	return fieldData, nil
}

// getSecretOverrides takes a Secret selector and returns the YAML data and handles k8s api errors appropriately
func getSecretOverrides(ctx spi.ComponentContext, selector *v1.SecretKeySelector) (string, error) {
	sec := &v1.Secret{}
	nsn := types.NamespacedName{Name: selector.Name, Namespace: ctx.EffectiveCR().Namespace}
	optional := selector.Optional
	err := ctx.Client().Get(context.TODO(), nsn, sec)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			ctx.Log().Errorf("Error retrieving Secret %s: %v", nsn.Name, err)
			return "", err
		}
		if optional == nil || !*optional {
			err = ctx.Log().ErrorfThrottledNewErr("Could not get Secret %s from namespace %s: %v", nsn.Name, nsn.Namespace, err)
			return "", err
		}
		ctx.Log().Infof("Optional Secret %s from namespace %s not found", nsn.Name, nsn.Namespace)
		return "", nil
	}

	dataStrings := map[string]string{}
	for key, val := range sec.Data {
		dataStrings[key] = string(val)
	}

	// Get resource data
	fieldData, ok := dataStrings[selector.Key]
	if !ok {
		if optional == nil || !*optional {
			err := ctx.Log().ErrorfThrottledNewErr("Could not get Data field %s from Resource %s from namespace %s", selector.Key, nsn.Name, nsn.Namespace)
			return "", err
		}
		ctx.Log().Infof("Optional Resource %s from namespace %s missing Data key %s", nsn.Name, nsn.Namespace, selector.Key)
	}
	return fieldData, nil
}
