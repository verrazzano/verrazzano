// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/override"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
)

var getControllerRuntimeClient = validators.GetClient

// CheckExternalIPsArgs method goes through the key-value pair to detect the presence of the
// specific key for the corresponding component that holds the external IP address
// It also checks whether IP addresses are valid and provided in a List format
func CheckExternalIPsArgs(installArgs []vzapi.InstallArgs, overrides []vzapi.Overrides, argsKeyName, jsonPath, compName, namespace string) error {
	var keyPresent bool
	var v1beta1Overrides = vzapi.ConvertValueOverridesToV1Beta1(overrides)

	c, err := getControllerRuntimeClient(runtime.NewScheme())
	if err != nil {
		return nil
	}
	overrideYAMLs, err := override.GetInstallOverridesYAMLUsingClient(c, v1beta1Overrides, namespace)
	for _, o := range overrideYAMLs {
		value, err := override.ExtractValueFromOverrideString(o, jsonPath)
		if err != nil {
			return err
		}
		v := castValuesToString(value)
		keyPresent = true
		if err := validateExternalIP(v, jsonPath, compName); err != nil {
			return err
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
func CheckExternalIPsOverridesArgs(overrides []v1beta1.Overrides, jsonPath, compName string, namespace string) error {
	c, err := getControllerRuntimeClient(runtime.NewScheme())
	if err != nil {
		return nil
	}
	overrideYAMLs, err := override.GetInstallOverridesYAMLUsingClient(c, overrides, namespace)
	for _, o := range overrideYAMLs {
		value, err := override.ExtractValueFromOverrideString(o, jsonPath)
		if err != nil {
			return err
		}
		v := castValuesToString(value)
		if err := validateExternalIP(v, jsonPath, compName); err != nil {
			return err
		}
	}
	return nil
}

// CheckExternalIPsOverridesArgsWithPaths method goes through the key-value pair to detect the presence of the
// specific keys for the corresponding component that holds the Service type and external IP address
// It checks whether the service is of a specific type.
// It also checks whether IP addresses are valid and provided in a List format
func CheckExternalIPsOverridesArgsWithPaths(overrides []v1beta1.Overrides, jsonBasePath, serviceTypePath, serviceTypeValue, externalIPPath, compName, namespace string) error {
	c, err := getControllerRuntimeClient(runtime.NewScheme())
	if err != nil {
		return nil
	}
	overrideYAMLs, err := override.GetInstallOverridesYAMLUsingClient(c, overrides, namespace)
	for _, o := range overrideYAMLs {
		value, err := override.ExtractValueFromOverrideString(o, jsonBasePath)
		if err != nil {
			return err
		}
		valueMap := value.(map[string]interface{})
		extractedIP := valueMap[externalIPPath]
		arrayOfExtractedIPs := castValuesToString(extractedIP)
		extractedType := valueMap[serviceTypePath]

		externalIPPathFull := jsonBasePath + "." + externalIPPath
		if extractedType != nil && extractedType == serviceTypeValue {
			if arrayOfExtractedIPs != nil {
				validateExternalIP(arrayOfExtractedIPs, externalIPPathFull, compName)
			}
		}
	}
	return nil
}

func validateExternalIP(addresses []string, key, compName string) error {
	if len(addresses) < 1 {
		return fmt.Errorf("At least one %s external IPs need to be set as an array for the key \"%v\"", compName, key)
	}
	for _, address := range addresses {
		if net.ParseIP(address) == nil {
			return fmt.Errorf("Controller external service key \"%v\" with IP \"%v\" is of invalid format for %s. Must be a proper IP address format", key, addresses[0], compName)
		}
	}
	return nil
}

func castValuesToString(value interface{}) []string {
	var values []string

	switch value.(type) {
	case string:
		values = append(values, value.(string))
	case []interface{}:
		valueArray := value.([]interface{})
		for _, v := range valueArray {
			values = append(values, v.(string))
		}
	}
	return values
}
