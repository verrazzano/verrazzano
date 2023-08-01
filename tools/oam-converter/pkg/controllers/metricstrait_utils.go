// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// source file: application-operator/controllers/metricstrait/metricstrait_utils.go
package controllers

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	operator "github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	"strings"
)
func CreateServiceMonitorName(trait *vzapi.MetricsTrait, appName string, compName string, portNum int) (string, error) {
	sname, err := createJobOrServiceMonitorName(trait, appName, compName, portNum)
	if err != nil {
		return "", err
	}
	return strings.Replace(sname, "_", "-", -1), nil
}
func createJobOrServiceMonitorName(trait *vzapi.MetricsTrait, appName string, compName string, portNum int) (string, error) {
	namespace := operator.GetNamespaceFromObjectMetaOrDefault(trait.ObjectMeta)
	if(types.InputArgs.Namespace != ""){
		namespace = types.InputArgs.Namespace
	}
	portStr := ""
	if portNum > 0 {
		portStr = fmt.Sprintf("_%d", portNum)
	}

	finalName := fmt.Sprintf("%s_%s_%s%s", appName, namespace, compName, portStr)
	// Check for Kubernetes name length requirement
	if len(finalName) > 63 {
		finalName = fmt.Sprintf("%s_%s%s", appName, namespace, portStr)
		if len(finalName) > 63 {
			return finalName[:63], nil
		}
	}
	return finalName, nil
}

