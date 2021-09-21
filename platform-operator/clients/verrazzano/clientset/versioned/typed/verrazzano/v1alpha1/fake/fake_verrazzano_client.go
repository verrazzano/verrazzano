// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned/typed/verrazzano/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeVerrazzanoV1alpha1 struct {
	*testing.Fake
}

func (c *FakeVerrazzanoV1alpha1) Verrazzanos(namespace string) v1alpha1.VerrazzanoInterface {
	return &FakeVerrazzanos{c, namespace}
}

func (c *FakeVerrazzanoV1alpha1) VerrazzanoComponents(namespace string) v1alpha1.VerrazzanoComponentInterface {
	return &FakeVerrazzanoComponents{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeVerrazzanoV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
