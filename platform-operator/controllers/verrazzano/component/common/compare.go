// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

func CompareInstallArgs(old []vzapi.InstallArgs, new []vzapi.InstallArgs, exceptions ...string) error {
	oldArgs := convertArgsToMap(old)
	newArgs := convertArgsToMap(new)
	for _, exception := range exceptions {
		delete(oldArgs, exception)
		delete(newArgs, exception)
	}
	if !reflect.DeepEqual(oldArgs, newArgs) {
		return fmt.Errorf("InstallArgs has been changed")
	}
	return nil
}

func convertArgsToMap(args []vzapi.InstallArgs) map[string]vzapi.InstallArgs {
	argsMap := make(map[string]vzapi.InstallArgs)
	for _, arg := range args {
		argsMap[arg.Name] = arg
	}
	return argsMap
}

func ComparePorts(old []corev1.ServicePort, new []corev1.ServicePort) error {
	oldPorts := convertPortsToMap(old)
	newPorts := convertPortsToMap(new)
	if !reflect.DeepEqual(oldPorts, newPorts) {
		return fmt.Errorf("ServicePort has been changed")
	}
	return nil
}

func convertPortsToMap(ports []corev1.ServicePort) map[int32]corev1.ServicePort {
	portMap := make(map[int32]corev1.ServicePort)
	for _, port := range ports {
		portMap[port.Port] = port
	}
	return portMap
}

func CompareInstallOverrides(old []v1beta1.Overrides, new []v1beta1.Overrides) error {
	for _, oldOverride := range old {
		for _, newOverride := range new {
			if err := compareJsonValues(*oldOverride.Values, *newOverride.Values); err != nil {
				return err
			}
		}
	}
	return nil
}

func compareJsonValues(old, new v1.JSON) error {
	var oldOverrideObj, newOverrideObj interface{}
	if err := json.Unmarshal(old.Raw, oldOverrideObj); err != nil {
		return fmt.Errorf("err %v, old JSON %s", err, old)
	}
	if err := json.Unmarshal(new.Raw, newOverrideObj); err != nil {
		return fmt.Errorf("err %v, new JSON %s", err, old)

	}
	if !reflect.DeepEqual(newOverrideObj, oldOverrideObj) {
		return fmt.Errorf("Old override JSON object: %v, new override JSON object: %v", oldOverrideObj, newOverrideObj)
	}
	return nil
}
