// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package v1beta2

import (
	"context"
	"time"

	v1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	scheme "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// PlatformsGetter has a method to return a PlatformInterface.
// A group's client should implement this interface.
type PlatformsGetter interface {
	Platforms(namespace string) PlatformInterface
}

// PlatformInterface has methods to work with Platform resources.
type PlatformInterface interface {
	Create(ctx context.Context, platform *v1beta2.Platform, opts v1.CreateOptions) (*v1beta2.Platform, error)
	Update(ctx context.Context, platform *v1beta2.Platform, opts v1.UpdateOptions) (*v1beta2.Platform, error)
	UpdateStatus(ctx context.Context, platform *v1beta2.Platform, opts v1.UpdateOptions) (*v1beta2.Platform, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta2.Platform, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta2.PlatformList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.Platform, err error)
	PlatformExpansion
}

// platforms implements PlatformInterface
type platforms struct {
	client rest.Interface
	ns     string
}

// newPlatforms returns a Platforms
func newPlatforms(c *VerrazzanoV1beta2Client, namespace string) *platforms {
	return &platforms{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the platform, and returns the corresponding platform object, and an error if there is any.
func (c *platforms) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta2.Platform, err error) {
	result = &v1beta2.Platform{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("platforms").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Platforms that match those selectors.
func (c *platforms) List(ctx context.Context, opts v1.ListOptions) (result *v1beta2.PlatformList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta2.PlatformList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("platforms").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested platforms.
func (c *platforms) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("platforms").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a platform and creates it.  Returns the server's representation of the platform, and an error, if there is any.
func (c *platforms) Create(ctx context.Context, platform *v1beta2.Platform, opts v1.CreateOptions) (result *v1beta2.Platform, err error) {
	result = &v1beta2.Platform{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("platforms").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(platform).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a platform and updates it. Returns the server's representation of the platform, and an error, if there is any.
func (c *platforms) Update(ctx context.Context, platform *v1beta2.Platform, opts v1.UpdateOptions) (result *v1beta2.Platform, err error) {
	result = &v1beta2.Platform{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("platforms").
		Name(platform.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(platform).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *platforms) UpdateStatus(ctx context.Context, platform *v1beta2.Platform, opts v1.UpdateOptions) (result *v1beta2.Platform, err error) {
	result = &v1beta2.Platform{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("platforms").
		Name(platform.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(platform).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the platform and deletes it. Returns an error if one occurs.
func (c *platforms) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("platforms").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *platforms) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("platforms").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched platform.
func (c *platforms) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.Platform, err error) {
	result = &v1beta2.Platform{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("platforms").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
