// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetCertificate returns the a cert in a given namespace for the cluster specified in the environment
func GetCertificate(namespace string, name string) (*unstructured.Unstructured, error) {
	client := GetDynamicClient()
	cert, err := client.Resource(getScheme()).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return cert, nil
}

// getScheme returns the WebLogic scheme needed to get unstructured data
func getScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "cert-manager.io",
		Version:  "v1alpha2",
		Resource: "certificate",
	}
}
