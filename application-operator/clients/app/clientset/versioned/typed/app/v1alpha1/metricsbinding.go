// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	scheme "github.com/verrazzano/verrazzano/application-operator/clients/app/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MetricsBindingsGetter has a method to return a MetricsBindingInterface.
// A group's client should implement this interface.
type MetricsBindingsGetter interface {
	MetricsBindings(namespace string) MetricsBindingInterface
}

// MetricsBindingInterface has methods to work with MetricsBinding resources.
type MetricsBindingInterface interface {
	Create(ctx context.Context, metricsBinding *v1alpha1.MetricsBinding, opts v1.CreateOptions) (*v1alpha1.MetricsBinding, error)
	Update(ctx context.Context, metricsBinding *v1alpha1.MetricsBinding, opts v1.UpdateOptions) (*v1alpha1.MetricsBinding, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.MetricsBinding, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.MetricsBindingList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MetricsBinding, err error)
	MetricsBindingExpansion
}

// metricsBindings implements MetricsBindingInterface
type metricsBindings struct {
	client rest.Interface
	ns     string
}

// newMetricsBindings returns a MetricsBindings
func newMetricsBindings(c *AppV1alpha1Client, namespace string) *metricsBindings {
	return &metricsBindings{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the metricsBinding, and returns the corresponding metricsBinding object, and an error if there is any.
func (c *metricsBindings) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MetricsBinding, err error) {
	result = &v1alpha1.MetricsBinding{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("metricsbindings").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MetricsBindings that match those selectors.
func (c *metricsBindings) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MetricsBindingList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MetricsBindingList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("metricsbindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested metricsBindings.
func (c *metricsBindings) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("metricsbindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a metricsBinding and creates it.  Returns the server's representation of the metricsBinding, and an error, if there is any.
func (c *metricsBindings) Create(ctx context.Context, metricsBinding *v1alpha1.MetricsBinding, opts v1.CreateOptions) (result *v1alpha1.MetricsBinding, err error) {
	result = &v1alpha1.MetricsBinding{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("metricsbindings").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(metricsBinding).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a metricsBinding and updates it. Returns the server's representation of the metricsBinding, and an error, if there is any.
func (c *metricsBindings) Update(ctx context.Context, metricsBinding *v1alpha1.MetricsBinding, opts v1.UpdateOptions) (result *v1alpha1.MetricsBinding, err error) {
	result = &v1alpha1.MetricsBinding{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("metricsbindings").
		Name(metricsBinding.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(metricsBinding).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the metricsBinding and deletes it. Returns an error if one occurs.
func (c *metricsBindings) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("metricsbindings").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *metricsBindings) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("metricsbindings").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched metricsBinding.
func (c *metricsBindings) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MetricsBinding, err error) {
	result = &v1alpha1.MetricsBinding{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("metricsbindings").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
