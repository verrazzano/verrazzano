// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package versioned

import (
	"fmt"
	"net/http"

	appv1alpha1 "github.com/verrazzano/verrazzano/application-operator/clientset/versioned/typed/app/v1alpha1"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/clientset/versioned/typed/clusters/v1alpha1"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/clientset/versioned/typed/oam/v1alpha1"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	AppV1alpha1() appv1alpha1.AppV1alpha1Interface
	ClustersV1alpha1() clustersv1alpha1.ClustersV1alpha1Interface
	OamV1alpha1() oamv1alpha1.OamV1alpha1Interface
}

// Clientset contains the clients for groups.
type Clientset struct {
	*discovery.DiscoveryClient
	appV1alpha1      *appv1alpha1.AppV1alpha1Client
	clustersV1alpha1 *clustersv1alpha1.ClustersV1alpha1Client
	oamV1alpha1      *oamv1alpha1.OamV1alpha1Client
}

// AppV1alpha1 retrieves the AppV1alpha1Client
func (c *Clientset) AppV1alpha1() appv1alpha1.AppV1alpha1Interface {
	return c.appV1alpha1
}

// ClustersV1alpha1 retrieves the ClustersV1alpha1Client
func (c *Clientset) ClustersV1alpha1() clustersv1alpha1.ClustersV1alpha1Interface {
	return c.clustersV1alpha1
}

// OamV1alpha1 retrieves the OamV1alpha1Client
func (c *Clientset) OamV1alpha1() oamv1alpha1.OamV1alpha1Interface {
	return c.oamV1alpha1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfig will generate a rate-limiter in configShallowCopy.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c

	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	// share the transport between all clients
	httpClient, err := rest.HTTPClientFor(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	return NewForConfigAndClient(&configShallowCopy, httpClient)
}

// NewForConfigAndClient creates a new Clientset for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
// If config's RateLimiter is not set and QPS and Burst are acceptable,
// NewForConfigAndClient will generate a rate-limiter in configShallowCopy.
func NewForConfigAndClient(c *rest.Config, httpClient *http.Client) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		if configShallowCopy.Burst <= 0 {
			return nil, fmt.Errorf("burst is required to be greater than 0 when RateLimiter is not set and QPS is set to greater than 0")
		}
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}

	var cs Clientset
	var err error
	cs.appV1alpha1, err = appv1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.clustersV1alpha1, err = clustersv1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	cs.oamV1alpha1, err = oamv1alpha1.NewForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfigAndClient(&configShallowCopy, httpClient)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	cs, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.appV1alpha1 = appv1alpha1.New(c)
	cs.clustersV1alpha1 = clustersv1alpha1.New(c)
	cs.oamV1alpha1 = oamv1alpha1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
