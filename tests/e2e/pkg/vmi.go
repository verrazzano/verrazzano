// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
)

// GetSystemVMICredentials - Obtain VMI system credentials
func GetSystemVMICredentials() (*UsernamePassword, error) {
	vmi, err := GetVerrazzanoMonitoringInstance("verrazzano-system", "system")
	if err != nil {
		return nil, fmt.Errorf("error getting system VMI: %w", err)
	}

	secret, err := GetSecret("verrazzano-system", vmi.Spec.SecretsName)
	if err != nil {
		return nil, err
	}

	username := secret.Data["username"]
	password := secret.Data["password"]
	if username == nil || password == nil {
		return nil, fmt.Errorf("username and password fields required in secret %v", secret)
	}

	return &UsernamePassword{
		Username: string(username),
		Password: string(password),
	}, nil
}

// GetBindingVmiHTTPClient returns the VMI client for the prided binding
func GetBindingVmiHTTPClient(bindingName string) *retryablehttp.Client {
	bindingVmiCaCert := getBindingVMICACert(bindingName)
	vmiRawClient := getHTTPClientWIthCABundle(bindingVmiCaCert)
	return newRetryableHTTPClient(vmiRawClient)
}

func getBindingVMICACert(bindingName string) []byte {
	return doGetCACertFromSecret(fmt.Sprintf("%v-tls", bindingName), "verrazzano-system")
}
