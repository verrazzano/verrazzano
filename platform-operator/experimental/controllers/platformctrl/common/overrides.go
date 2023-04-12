// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package common

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/override"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConvertToHelmOverrides Builds the helm overrides for a release, including image and file, and custom overrides
// - returns an error and a HelmOverride struct with the field populated
func ConvertToHelmOverrides(log vzlog.VerrazzanoLogger, client client.Client, releaseName string, namespace string, modOverrides []v1beta2.Overrides) ([]helm.HelmOverrides, error) {
	var kvs []bom.KeyValue
	var err error
	var overrides []helm.HelmOverrides

	// Sort the kvs list by priority (0th term has the highest priority)

	defer vzos.RemoveTempFiles(log.GetRootZapLogger(), fmt.Sprintf(`helm-overrides-user-%s.*\.yaml`, releaseName))

	// Getting user defined Helm overrides as the highest priority
	overrideStrings, err := override.GetInstallOverridesYAMLUsingClient(client, convertToV1Beta1Overrides(modOverrides), namespace)
	if err != nil {
		return overrides, err
	}
	for _, overrideString := range overrideStrings {
		file, err := vzos.CreateTempFile(fmt.Sprintf("helm-overrides-user-%s-*.yaml", releaseName), []byte(overrideString))
		if err != nil {
			log.Error(err.Error())
			return overrides, err
		}
		kvs = append(kvs, bom.KeyValue{Value: file.Name(), IsFile: true})
	}

	// Convert the key value pairs to Helm overrides
	overrides = organizeHelmOverrides(kvs)
	return overrides, nil
}

func convertToV1Beta1Overrides(overrides []v1beta2.Overrides) []v1beta1.Overrides {
	var convertedOverrides []v1beta1.Overrides
	for _, override := range overrides {
		convertedOverrides = append(convertedOverrides, v1beta1.Overrides{
			ConfigMapRef: override.ConfigMapRef.DeepCopy(),
			SecretRef:    override.SecretRef.DeepCopy(),
			Values:       override.Values.DeepCopy(),
		})
	}
	return convertedOverrides
}

// organizeHelmOverrides creates a list of Helm overrides from key value pairs in reverse precedence (0th value has the lowest precedence)
// Each key value pair gets its own override object to keep strict precedence
func organizeHelmOverrides(kvs []bom.KeyValue) []helm.HelmOverrides {
	var overrides []helm.HelmOverrides
	for _, kv := range kvs {
		if kv.SetString {
			// Append in reverse order because helm precedence is right to left
			overrides = append([]helm.HelmOverrides{{SetStringOverrides: fmt.Sprintf("%s=%s", kv.Key, kv.Value)}}, overrides...)
		} else if kv.SetFile {
			// Append in reverse order because helm precedence is right to left
			overrides = append([]helm.HelmOverrides{{SetFileOverrides: fmt.Sprintf("%s=%s", kv.Key, kv.Value)}}, overrides...)
		} else if kv.IsFile {
			// Append in reverse order because helm precedence is right to left
			overrides = append([]helm.HelmOverrides{{FileOverride: kv.Value}}, overrides...)
		} else {
			// Append in reverse order because helm precedence is right to left
			overrides = append([]helm.HelmOverrides{{SetOverrides: fmt.Sprintf("%s=%s", kv.Key, kv.Value)}}, overrides...)
		}
	}
	return overrides
}
