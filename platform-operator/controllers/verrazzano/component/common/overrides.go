// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"

	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

// GetInstallOverridesYAML takes the list of Overrides and returns a string array of YAMLs
func GetInstallOverridesYAML(ctx spi.ComponentContext, overrides []v1alpha1.Overrides) ([]string, error) {
	return getInstallOverridesYAML(ctx.Log(), ctx.Client(), v1alpha1.ConvertValueOverridesToV1Beta1(overrides), ctx.EffectiveCR().Namespace)
}

// GetInstallOverridesYAMLUsingClient takes the list of Overrides and returns a string array of YAMLs using the
// specified client
func GetInstallOverridesYAMLUsingClient(client client.Client, overrides []v1beta1.Overrides, namespace string) ([]string, error) {
	// DefaultLogger is used since this is invoked from validateInstall and validateUpdate functions and
	// any actual logging isn't being performed
	log := vzlog.DefaultLogger()
	return getInstallOverridesYAML(log, client, overrides, namespace)
}

// ExtractValueFromOverrideString is a helper function to extract a given value from override.
func ExtractValueFromOverrideString(overrideStr string, field string) (interface{}, error) {
	jsonConfig, err := yaml.YAMLToJSON([]byte(overrideStr))
	if err != nil {
		return nil, err
	}
	jsonString, err := gabs.ParseJSON(jsonConfig)
	if err != nil {
		return nil, err
	}
	return jsonString.Path(field).Data(), nil
}

// getInstallOverridesYAML takes the list of Overrides and returns a string array of YAMLs
func getInstallOverridesYAML(log vzlog.VerrazzanoLogger, client client.Client, overrides []v1beta1.Overrides,
	namespace string) ([]string, error) {
	var overrideStrings []string
	for _, override := range overrides {
		// Check if ConfigMapRef is populated and gather data
		if override.ConfigMapRef != nil {
			// Get the ConfigMap data
			data, err := getConfigMapOverrides(log, client, override.ConfigMapRef, namespace)
			if err != nil {
				return overrideStrings, err
			}
			overrideStrings = append(overrideStrings, data)
			continue
		}
		// Check if SecretRef is populated and gather data
		if override.SecretRef != nil {
			// Get the Secret data
			data, err := getSecretOverrides(log, client, override.SecretRef, namespace)
			if err != nil {
				return overrideStrings, err
			}
			overrideStrings = append(overrideStrings, data)
			continue
		}
		if override.Values != nil {
			overrideValuesData, err := yaml.Marshal(override.Values)
			if err != nil {
				return overrideStrings, err
			}
			overrideStrings = append(overrideStrings, string(overrideValuesData))
		}

	}
	return overrideStrings, nil
}

// getConfigMapOverrides takes a ConfigMap selector and returns the YAML data and handles k8s api errors appropriately
func getConfigMapOverrides(log vzlog.VerrazzanoLogger, client client.Client, selector *v1.ConfigMapKeySelector,
	namespace string) (string, error) {
	configMap := &v1.ConfigMap{}
	nsn := types.NamespacedName{Name: selector.Name, Namespace: namespace}
	optional := selector.Optional
	err := client.Get(context.TODO(), nsn, configMap)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			log.Errorf("Error retrieving ConfigMap %s: %v", nsn.Name, err)
			return "", err
		}
		if optional == nil || !*optional {
			err = log.ErrorfThrottledNewErr("Could not get Configmap %s from namespace %s: %v", nsn.Name, nsn.Namespace, err)
			return "", err
		}
		log.Infof("Optional Configmap %s from namespace %s not found", nsn.Name, nsn.Namespace)
		return "", nil
	}

	// Get resource data
	fieldData, ok := configMap.Data[selector.Key]
	if !ok {
		if optional == nil || !*optional {
			err := log.ErrorfThrottledNewErr("Could not get Data field %s from Resource %s from namespace %s", selector.Key, nsn.Name, nsn.Namespace)
			return "", err
		}
		log.Infof("Optional Resource %s from namespace %s missing Data key %s", nsn.Name, nsn.Namespace, selector.Key)
	}
	return fieldData, nil
}

// getSecretOverrides takes a Secret selector and returns the YAML data and handles k8s api errors appropriately
func getSecretOverrides(log vzlog.VerrazzanoLogger, client client.Client, selector *v1.SecretKeySelector,
	namespace string) (string, error) {
	sec := &v1.Secret{}
	nsn := types.NamespacedName{Name: selector.Name, Namespace: namespace}
	optional := selector.Optional
	err := client.Get(context.TODO(), nsn, sec)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			log.Errorf("Error retrieving Secret %s: %v", nsn.Name, err)
			return "", err
		}
		if optional == nil || !*optional {
			err = log.ErrorfThrottledNewErr("Could not get Secret %s from namespace %s: %v", nsn.Name, nsn.Namespace, err)
			return "", err
		}
		log.Infof("Optional Secret %s from namespace %s not found", nsn.Name, nsn.Namespace)
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
			err := log.ErrorfThrottledNewErr("Could not get Data field %s from Resource %s from namespace %s", selector.Key, nsn.Name, nsn.Namespace)
			return "", err
		}
		log.Infof("Optional Resource %s from namespace %s missing Data key %s", nsn.Name, nsn.Namespace, selector.Key)
	}
	return fieldData, nil
}
