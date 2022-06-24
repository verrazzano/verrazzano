// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzk8sfake

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	restclient "k8s.io/client-go/rest"
	fakeRESTClient "k8s.io/client-go/rest/fake"
)

func NewClientsetConfig(objects ...runtime.Object) (*restclient.Config, kubernetes.Interface) {
	cfg, _ := restclient.InClusterConfig()
	client := fakeclientset.NewSimpleClientset(objects...)
	wrappedClient := &WrappedClientset{Clientset: client}
	return cfg, wrappedClient
}

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
		RestClient: &fakeRESTClient.RESTClient{},
	}
}

func (w *WrappedCoreV1) RESTClient() restclient.Interface {
	return w.RestClient
}
