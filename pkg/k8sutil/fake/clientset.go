// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fake

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	restclient "k8s.io/client-go/rest"
	fakeRESTClient "k8s.io/client-go/rest/fake"
)

func NewClientsetConfig(objects ...runtime.Object) (*restclient.Config, kubernetes.Interface) {
	cfg := &restclient.Config{}
	client := fakeclientset.NewSimpleClientset(objects...)
	wrappedClient := &WrappedClientset{Clientset: client}
	return cfg, wrappedClient
}

/*
There is a deficiency in the FakeCoreV1 implementation that returns a nil RESTClient
To get around this limitation, we wrap the fakes in our own types, which supply a valid
RESTClient for unit testing.
*/
type (
	WrappedClientset struct {
		*fakeclientset.Clientset
	}
	WrappedCoreV1 struct {
		fakecorev1.FakeCoreV1
		RestClient restclient.Interface
	}
)

func (w *WrappedClientset) CoreV1() corev1.CoreV1Interface {
	return &WrappedCoreV1{
		FakeCoreV1: fakecorev1.FakeCoreV1{
			Fake: &w.Fake,
		},
		RestClient: &fakeRESTClient.RESTClient{GroupVersion: schema.GroupVersion{Version: "v1"}},
	}
}

func (w *WrappedCoreV1) RESTClient() restclient.Interface {
	return w.RestClient
}
