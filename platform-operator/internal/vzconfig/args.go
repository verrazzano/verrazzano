// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"net"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

// CheckExternalIPsArgs method goes through the key-value pair to detect the presence of the
// specific key for the corresponding component that holds the external IP address
// It also checks whether IP addresses are valid and provided in a List format
func CheckExternalIPsArgs(installArgs []vzapi.InstallArgs, overrides []vzapi.Overrides, argsKeyName, jsonPath, compName string) error {
	var keyPresent bool
	for _, override := range overrides {
		o, err := gabs.ParseJSON(override.Values.Raw)
		if err != nil {
			return err
		}
		if container := o.Path(jsonPath); container != nil {
			keyPresent = true
			if err := validateExternalIP([]string{container.Data().(string)}, jsonPath, compName); err != nil {
				return err
			}
		}
	}
	if keyPresent {
		return nil
	}

	for _, installArg := range installArgs {
		if installArg.Name == argsKeyName {
			keyPresent = true
			if err := validateExternalIP(installArg.ValueList, installArg.Name, compName); err != nil {
				return err
			}
		}
	}
	if !keyPresent {
		return fmt.Errorf("Key \"%v\" not found for component \"%v\" for type NodePort", argsKeyName, compName)
	}
	return nil
}

func validateExternalIP(addresses []string, key, compName string) error {
	if len(addresses) < 1 {
		return fmt.Errorf("At least one %s external IPs need to be set as an array for the key \"%v\"", compName, key)
	}
	if net.ParseIP(addresses[0]) == nil {
		return fmt.Errorf("Controller external service key \"%v\" with IP \"%v\" is of invalid format for %s. Must be a proper IP address format", key, addresses[0], compName)
	}
	return nil
}
