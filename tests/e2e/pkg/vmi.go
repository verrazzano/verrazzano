// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"

	"github.com/hashicorp/go-retryablehttp"
)

// GetSystemVMICredentials - Obtain VMI system credentials
func GetSystemVMICredentials() (*UsernamePassword, error) {
	secret, err := GetSecret("verrazzano-system", "verrazzano")
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
func GetBindingVmiHTTPClient(bindingName string, kubeconfigPath string) (*retryablehttp.Client, error) {
	bindingVmiCaCert, err := getBindingVMICACert(bindingName, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	vmiRawClient := getHTTPClientWithCABundle(bindingVmiCaCert, kubeconfigPath)
	retryableClient := newRetryableHTTPClient(vmiRawClient)
	retryableClient.CheckRetry = GetRetryPolicy()
	return retryableClient, nil
}

func getBindingVMICACert(bindingName string, kubeconfigPath string) ([]byte, error) {
	return doGetCACertFromSecret(fmt.Sprintf("%v-tls", bindingName), "verrazzano-system", kubeconfigPath)
}
