// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1beta1 "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned/typed/verrazzano/v1beta1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeVerrazzanoV1beta1 struct {
	*testing.Fake
}

func (c *FakeVerrazzanoV1beta1) Verrazzanos(namespace string) v1beta1.VerrazzanoInterface {
	return &FakeVerrazzanos{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeVerrazzanoV1beta1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
