// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/controllers/appconfig"

	"github.com/verrazzano/verrazzano/application-operator/constants"
)

// ConvertAPIVersionToGroupAndVersion splits APIVersion into API and version parts.
// An APIVersion takes the form api/version (e.g. networking.k8s.io/v1)
// If the input does not contain a / the group is defaulted to the empty string.
// apiVersion - The combined api and version to split
func ConvertAPIVersionToGroupAndVersion(apiVersion string) (string, string) {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) < 2 {
		// Use empty group for core types.
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// IsWorkloadMarkedForUpgrade checks to see if a workload needs to be upgraded to the latest
// Verrazzano version. Verrazzano defines some resources which are used by applications and when Verrazzano is upgraded,
// a user can mark an application to indicate that it should use the latest resources defined by Verrazzano.
// A response of 'true' indicates that reconcile should use the latest resources defined by Verrazzano and a response
// of 'false' indicates that the application should continue to use the current values.
func IsWorkloadMarkedForUpgrade(annotations map[string]string, previousUpgrade string) bool {
	return annotations[constants.AnnotationUpgradeVersion] != previousUpgrade
}

// IsWorkloadMarkedForRestart checks to see if a workload needs to be restarted.
func IsWorkloadMarkedForRestart(annotations map[string]string, previousRestartVersion string, log logr.Logger) bool {
	toRestart := annotations[appconfig.RestartVersionAnnotation] != previousRestartVersion
	if toRestart {
		log.Info(fmt.Sprintf("The workload is marked for restart since the restart version has been changed from %s to %s",
			previousRestartVersion, annotations[appconfig.RestartVersionAnnotation]))
	}
	return toRestart
}
