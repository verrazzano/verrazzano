// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	"fmt"
	"net"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

// This method goes through the key-value pair to detect the presence of the
// specific key for the corresponding component that holds the external IP address
// It also checks whether IP addresses are valid and provided in a List format
func CheckExternalIPsArgs(installArgs []vzapi.InstallArgs, argsKeyName, compName string) error {
	var keyPresent bool
	for _, installArg := range installArgs {
		if installArg.Name == argsKeyName {
			keyPresent = true
			if len(installArg.ValueList) < 1 {
				return fmt.Errorf("At least one %s external IPs need to be set as an array for the key \"%v\"", compName, installArg.Name)
			}
			if net.ParseIP(installArg.ValueList[0]) == nil {
				return fmt.Errorf("Controller external service key \"%v\" with IP \"%v\" is of invalid format for %s. Must be a proper IP address format", installArg.Name, installArg.ValueList[0], compName)
			}
		}
	}
	if !keyPresent {
		return fmt.Errorf("Key \"%v\" not found for component \"%v\" for type NodePort", argsKeyName, compName)
	}
	return nil
}
