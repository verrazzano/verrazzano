// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const (
	MysqlInstallValuesWebhook      = "verrazzano-platform-mysqlinstalloverrides"
	MysqlInstallValuesV1beta1path  = "/v1beta1-validate-mysql-install-override-values"
	MysqlInstallValuesV1alpha1path = "/v1alpha1-validate-mysql-install-override-values"
	RequirementsWebhook            = "verrazzano-platform-requirements-validator"
	RequirementsV1beta1Path        = "/v1beta1-validate-requirements"
	RequirementsV1alpha1Path       = "/v1alpha1-validate-requirements"
)

// isMinVersion indicates whether the provide version is greater than the min version provided
func isMinVersion(vzVersion, minVersion string) bool {
	vzSemver, err := semver.NewSemVersion(vzVersion)
	if err != nil {
		return false
	}
	minSemver, err := semver.NewSemVersion(minVersion)
	if err != nil {
		return false
	}
	return !vzSemver.IsLessThan(minSemver)
}

// extractValueFromOverrideString extracts  a given value from override.
func extractValueFromOverrideString(overrideStr string, field string) (interface{}, error) {
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

// ValidatePlatformOperatorSingleton iterates over the list of pods and verifies that there is only a single VPO instance running
func ValidatePlatformOperatorSingleton(podList v1.PodList) error {
	if len(podList.Items) > 1 {
		healthyPod := 0
		for _, pod := range podList.Items {
			if pod.Status.Phase != "Failed" {
				healthyPod++
			}
		}
		if healthyPod > 1 {
			return fmt.Errorf("Found more than one running instance of the platform operator, only one instance allowed")
		}
	}
	return nil
}
