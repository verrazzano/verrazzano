// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/clientset/versioned/typed/oam/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeOamV1alpha1 struct {
	*testing.Fake
}

func (c *FakeOamV1alpha1) IngressTraits(namespace string) v1alpha1.IngressTraitInterface {
	return &FakeIngressTraits{c, namespace}
}

func (c *FakeOamV1alpha1) LoggingTraits(namespace string) v1alpha1.LoggingTraitInterface {
	return &FakeLoggingTraits{c, namespace}
}

func (c *FakeOamV1alpha1) MetricsTraits(namespace string) v1alpha1.MetricsTraitInterface {
	return &FakeMetricsTraits{c, namespace}
}

func (c *FakeOamV1alpha1) VerrazzanoCoherenceWorkloads(namespace string) v1alpha1.VerrazzanoCoherenceWorkloadInterface {
	return &FakeVerrazzanoCoherenceWorkloads{c, namespace}
}

func (c *FakeOamV1alpha1) VerrazzanoHelidonWorkloads(namespace string) v1alpha1.VerrazzanoHelidonWorkloadInterface {
	return &FakeVerrazzanoHelidonWorkloads{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeOamV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
