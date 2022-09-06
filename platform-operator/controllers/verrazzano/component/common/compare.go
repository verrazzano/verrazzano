// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"reflect"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
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
