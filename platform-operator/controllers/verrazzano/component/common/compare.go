// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"reflect"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

func CompareInstallArgs(old []vzapi.InstallArgs, new []vzapi.InstallArgs, exceptions []string) error {
	oldArgs := convertArgsToMap(old)
	newArgs := convertArgsToMap(new)
	if exceptions != nil {
		for _, exception := range exceptions {
			delete(oldArgs, exception)
			delete(newArgs, exception)
		}
	}
	if !reflect.DeepEqual(oldArgs, newArgs) {
		return fmt.Errorf("InstallArgs has been changed")
	}
	return nil
}

func convertArgsToMap(args []vzapi.InstallArgs) map[string]vzapi.InstallArgs {
	argsMap := make(map[string]vzapi.InstallArgs)
	if args != nil {
		for _, arg := range args {
			argsMap[arg.Name] = arg
		}
	}
	return argsMap
}
