// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/clients/app/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type AppV1alpha1Interface interface {
	RESTClient() rest.Interface
	MetricsBindingsGetter
	MetricsTemplatesGetter
}

// AppV1alpha1Client is used to interact with features provided by the app.verrazzano.io group.
type AppV1alpha1Client struct {
	restClient rest.Interface
}

func (c *AppV1alpha1Client) MetricsBindings(namespace string) MetricsBindingInterface {
	return newMetricsBindings(c, namespace)
}

func (c *AppV1alpha1Client) MetricsTemplates(namespace string) MetricsTemplateInterface {
	return newMetricsTemplates(c, namespace)
}

// NewForConfig creates a new AppV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*AppV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &AppV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new AppV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *AppV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new AppV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *AppV1alpha1Client {
	return &AppV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *AppV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
