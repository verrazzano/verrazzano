// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	scheme "github.com/verrazzano/verrazzano/application-operator/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MultiClusterConfigMapsGetter has a method to return a MultiClusterConfigMapInterface.
// A group's client should implement this interface.
type MultiClusterConfigMapsGetter interface {
	MultiClusterConfigMaps(namespace string) MultiClusterConfigMapInterface
}

// MultiClusterConfigMapInterface has methods to work with MultiClusterConfigMap resources.
type MultiClusterConfigMapInterface interface {
	Create(ctx context.Context, multiClusterConfigMap *v1alpha1.MultiClusterConfigMap, opts v1.CreateOptions) (*v1alpha1.MultiClusterConfigMap, error)
	Update(ctx context.Context, multiClusterConfigMap *v1alpha1.MultiClusterConfigMap, opts v1.UpdateOptions) (*v1alpha1.MultiClusterConfigMap, error)
	UpdateStatus(ctx context.Context, multiClusterConfigMap *v1alpha1.MultiClusterConfigMap, opts v1.UpdateOptions) (*v1alpha1.MultiClusterConfigMap, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.MultiClusterConfigMap, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.MultiClusterConfigMapList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MultiClusterConfigMap, err error)
	MultiClusterConfigMapExpansion
}

// multiClusterConfigMaps implements MultiClusterConfigMapInterface
type multiClusterConfigMaps struct {
	client rest.Interface
	ns     string
}

// newMultiClusterConfigMaps returns a MultiClusterConfigMaps
func newMultiClusterConfigMaps(c *ClustersV1alpha1Client, namespace string) *multiClusterConfigMaps {
	return &multiClusterConfigMaps{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the multiClusterConfigMap, and returns the corresponding multiClusterConfigMap object, and an error if there is any.
func (c *multiClusterConfigMaps) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MultiClusterConfigMap, err error) {
	result = &v1alpha1.MultiClusterConfigMap{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("multiclusterconfigmaps").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MultiClusterConfigMaps that match those selectors.
func (c *multiClusterConfigMaps) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MultiClusterConfigMapList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MultiClusterConfigMapList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("multiclusterconfigmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested multiClusterConfigMaps.
func (c *multiClusterConfigMaps) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("multiclusterconfigmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a multiClusterConfigMap and creates it.  Returns the server's representation of the multiClusterConfigMap, and an error, if there is any.
func (c *multiClusterConfigMaps) Create(ctx context.Context, multiClusterConfigMap *v1alpha1.MultiClusterConfigMap, opts v1.CreateOptions) (result *v1alpha1.MultiClusterConfigMap, err error) {
	result = &v1alpha1.MultiClusterConfigMap{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("multiclusterconfigmaps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(multiClusterConfigMap).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a multiClusterConfigMap and updates it. Returns the server's representation of the multiClusterConfigMap, and an error, if there is any.
func (c *multiClusterConfigMaps) Update(ctx context.Context, multiClusterConfigMap *v1alpha1.MultiClusterConfigMap, opts v1.UpdateOptions) (result *v1alpha1.MultiClusterConfigMap, err error) {
	result = &v1alpha1.MultiClusterConfigMap{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("multiclusterconfigmaps").
		Name(multiClusterConfigMap.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(multiClusterConfigMap).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *multiClusterConfigMaps) UpdateStatus(ctx context.Context, multiClusterConfigMap *v1alpha1.MultiClusterConfigMap, opts v1.UpdateOptions) (result *v1alpha1.MultiClusterConfigMap, err error) {
	result = &v1alpha1.MultiClusterConfigMap{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("multiclusterconfigmaps").
		Name(multiClusterConfigMap.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(multiClusterConfigMap).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the multiClusterConfigMap and deletes it. Returns an error if one occurs.
func (c *multiClusterConfigMaps) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("multiclusterconfigmaps").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *multiClusterConfigMaps) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("multiclusterconfigmaps").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched multiClusterConfigMap.
func (c *multiClusterConfigMaps) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MultiClusterConfigMap, err error) {
	result = &v1alpha1.MultiClusterConfigMap{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("multiclusterconfigmaps").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
