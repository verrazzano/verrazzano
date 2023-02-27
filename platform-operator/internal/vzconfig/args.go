// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"net"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

// CheckExternalIPsArgs method goes through the key-value pair to detect the presence of the
// specific key for the corresponding component that holds the external IP address
// It also checks whether IP addresses are valid and provided in a List format
func CheckExternalIPsArgs(installArgs []vzapi.InstallArgs, overrides []vzapi.Overrides, argsKeyName, jsonPath, compName string) error {
	var keyPresent bool
	for i, override := range overrides {
		if err := checkIfConfigMapOrSecret(overrides, i); err != nil {
			return err
		}
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

// CheckExternalIPsOverridesArgs method goes through the key-value pair to detect the presence of the
// specific key for the corresponding component that holds the external IP address
// It also checks whether IP addresses are valid and provided in a List format
func CheckExternalIPsOverridesArgs(overrides []v1beta1.Overrides, jsonPath, compName string) error {
	for _, override := range overrides {
		o, err := gabs.ParseJSON(override.Values.Raw)
		if err != nil {
			return err
		}
		if container := o.Path(jsonPath); container != nil {
			if err := validateExternalIP([]string{container.Data().(string)}, jsonPath, compName); err != nil {
				return err
			}
		}
	}
	return nil
}

// CheckExternalIPsOverridesArgsWithPaths method goes through the key-value pair to detect the presence of the
// specific keys for the corresponding component that holds the Service type and external IP address
// It checks whether the service is of a specific type.
// It also checks whether IP addresses are valid and provided in a List format
func CheckExternalIPsOverridesArgsWithPaths(overrides []v1beta1.Overrides, jsonBasePath, serviceTypePath, serviceTypeValue, externalIPPath, compName string) error {
	for _, override := range overrides {
		o, err := gabs.ParseJSON(override.Values.Raw)
		if err != nil {
			return err
		}
		typePathFull := jsonBasePath + "." + serviceTypePath
		externalIPPathFull := jsonBasePath + "." + externalIPPath
		if typePathContainer := o.Path(typePathFull); typePathContainer != nil && typePathContainer.Data().(string) == serviceTypeValue {
			if externalIPPathContainer := o.Path(externalIPPathFull); externalIPPathContainer != nil {
				if err := validateExternalIP([]string{externalIPPathContainer.Data().(string)}, externalIPPathFull, compName); err != nil {
					return err
				}
			}
		}
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

func checkIfConfigMapOrSecret(overrides []vzapi.Overrides, override int) error {
	if _, err := gabs.ParseJSON([]byte(overrides[override].ConfigMapRef.Name)); err != nil {
		return err
	}

	return nil
}
