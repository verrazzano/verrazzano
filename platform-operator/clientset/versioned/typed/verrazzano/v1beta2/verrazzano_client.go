// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package v1beta2

import (
	"net/http"

	v1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	"github.com/verrazzano/verrazzano/platform-operator/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type VerrazzanoV1beta2Interface interface {
	RESTClient() rest.Interface
	ModulesGetter
	ModuleLifecyclesGetter
}

// VerrazzanoV1beta2Client is used to interact with features provided by the verrazzano group.
type VerrazzanoV1beta2Client struct {
	restClient rest.Interface
}

func (c *VerrazzanoV1beta2Client) Modules(namespace string) ModuleInterface {
	return newModules(c, namespace)
}

func (c *VerrazzanoV1beta2Client) ModuleLifecycles(namespace string) ModuleLifecycleInterface {
	return newModuleLifecycles(c, namespace)
}

// NewForConfig creates a new VerrazzanoV1beta2Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*VerrazzanoV1beta2Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new VerrazzanoV1beta2Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*VerrazzanoV1beta2Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &VerrazzanoV1beta2Client{client}, nil
}

// NewForConfigOrDie creates a new VerrazzanoV1beta2Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *VerrazzanoV1beta2Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new VerrazzanoV1beta2Client for the given RESTClient.
func New(c rest.Interface) *VerrazzanoV1beta2Client {
	return &VerrazzanoV1beta2Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1beta2.SchemeGroupVersion
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
func (c *VerrazzanoV1beta2Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
