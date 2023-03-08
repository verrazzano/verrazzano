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

// VerrazzanoModulesGetter has a method to return a VerrazzanoModuleInterface.
// A group's client should implement this interface.
type VerrazzanoModulesGetter interface {
	VerrazzanoModules(namespace string) VerrazzanoModuleInterface
}

// VerrazzanoModuleInterface has methods to work with VerrazzanoModule resources.
type VerrazzanoModuleInterface interface {
	Create(ctx context.Context, verrazzanoModule *v1beta2.VerrazzanoModule, opts v1.CreateOptions) (*v1beta2.VerrazzanoModule, error)
	Update(ctx context.Context, verrazzanoModule *v1beta2.VerrazzanoModule, opts v1.UpdateOptions) (*v1beta2.VerrazzanoModule, error)
	UpdateStatus(ctx context.Context, verrazzanoModule *v1beta2.VerrazzanoModule, opts v1.UpdateOptions) (*v1beta2.VerrazzanoModule, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta2.VerrazzanoModule, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta2.VerrazzanoModuleList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.VerrazzanoModule, err error)
	VerrazzanoModuleExpansion
}

// verrazzanoModules implements VerrazzanoModuleInterface
type verrazzanoModules struct {
	client rest.Interface
	ns     string
}

// newVerrazzanoModules returns a VerrazzanoModules
func newVerrazzanoModules(c *VerrazzanoV1beta2Client, namespace string) *verrazzanoModules {
	return &verrazzanoModules{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the verrazzanoModule, and returns the corresponding verrazzanoModule object, and an error if there is any.
func (c *verrazzanoModules) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta2.VerrazzanoModule, err error) {
	result = &v1beta2.VerrazzanoModule{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("verrazzanomodules").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of VerrazzanoModules that match those selectors.
func (c *verrazzanoModules) List(ctx context.Context, opts v1.ListOptions) (result *v1beta2.VerrazzanoModuleList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta2.VerrazzanoModuleList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("verrazzanomodules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested verrazzanoModules.
func (c *verrazzanoModules) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("verrazzanomodules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a verrazzanoModule and creates it.  Returns the server's representation of the verrazzanoModule, and an error, if there is any.
func (c *verrazzanoModules) Create(ctx context.Context, verrazzanoModule *v1beta2.VerrazzanoModule, opts v1.CreateOptions) (result *v1beta2.VerrazzanoModule, err error) {
	result = &v1beta2.VerrazzanoModule{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("verrazzanomodules").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(verrazzanoModule).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a verrazzanoModule and updates it. Returns the server's representation of the verrazzanoModule, and an error, if there is any.
func (c *verrazzanoModules) Update(ctx context.Context, verrazzanoModule *v1beta2.VerrazzanoModule, opts v1.UpdateOptions) (result *v1beta2.VerrazzanoModule, err error) {
	result = &v1beta2.VerrazzanoModule{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("verrazzanomodules").
		Name(verrazzanoModule.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(verrazzanoModule).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *verrazzanoModules) UpdateStatus(ctx context.Context, verrazzanoModule *v1beta2.VerrazzanoModule, opts v1.UpdateOptions) (result *v1beta2.VerrazzanoModule, err error) {
	result = &v1beta2.VerrazzanoModule{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("verrazzanomodules").
		Name(verrazzanoModule.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(verrazzanoModule).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the verrazzanoModule and deletes it. Returns an error if one occurs.
func (c *verrazzanoModules) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("verrazzanomodules").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *verrazzanoModules) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("verrazzanomodules").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched verrazzanoModule.
func (c *verrazzanoModules) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.VerrazzanoModule, err error) {
	result = &v1beta2.VerrazzanoModule{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("verrazzanomodules").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
