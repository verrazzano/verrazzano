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

// ModuleDefinitionsGetter has a method to return a ModuleDefinitionInterface.
// A group's client should implement this interface.
type ModuleDefinitionsGetter interface {
	ModuleDefinitions() ModuleDefinitionInterface
}

// ModuleDefinitionInterface has methods to work with ModuleDefinition resources.
type ModuleDefinitionInterface interface {
	Create(ctx context.Context, moduleDefinition *v1beta2.ModuleDefinition, opts v1.CreateOptions) (*v1beta2.ModuleDefinition, error)
	Update(ctx context.Context, moduleDefinition *v1beta2.ModuleDefinition, opts v1.UpdateOptions) (*v1beta2.ModuleDefinition, error)
	UpdateStatus(ctx context.Context, moduleDefinition *v1beta2.ModuleDefinition, opts v1.UpdateOptions) (*v1beta2.ModuleDefinition, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1beta2.ModuleDefinition, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1beta2.ModuleDefinitionList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.ModuleDefinition, err error)
	ModuleDefinitionExpansion
}

// moduleDefinitions implements ModuleDefinitionInterface
type moduleDefinitions struct {
	client rest.Interface
}

// newModuleDefinitions returns a ModuleDefinitions
func newModuleDefinitions(c *VerrazzanoV1beta2Client) *moduleDefinitions {
	return &moduleDefinitions{
		client: c.RESTClient(),
	}
}

// Get takes name of the moduleDefinition, and returns the corresponding moduleDefinition object, and an error if there is any.
func (c *moduleDefinitions) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta2.ModuleDefinition, err error) {
	result = &v1beta2.ModuleDefinition{}
	err = c.client.Get().
		Resource("moduledefinitions").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ModuleDefinitions that match those selectors.
func (c *moduleDefinitions) List(ctx context.Context, opts v1.ListOptions) (result *v1beta2.ModuleDefinitionList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1beta2.ModuleDefinitionList{}
	err = c.client.Get().
		Resource("moduledefinitions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested moduleDefinitions.
func (c *moduleDefinitions) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("moduledefinitions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a moduleDefinition and creates it.  Returns the server's representation of the moduleDefinition, and an error, if there is any.
func (c *moduleDefinitions) Create(ctx context.Context, moduleDefinition *v1beta2.ModuleDefinition, opts v1.CreateOptions) (result *v1beta2.ModuleDefinition, err error) {
	result = &v1beta2.ModuleDefinition{}
	err = c.client.Post().
		Resource("moduledefinitions").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(moduleDefinition).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a moduleDefinition and updates it. Returns the server's representation of the moduleDefinition, and an error, if there is any.
func (c *moduleDefinitions) Update(ctx context.Context, moduleDefinition *v1beta2.ModuleDefinition, opts v1.UpdateOptions) (result *v1beta2.ModuleDefinition, err error) {
	result = &v1beta2.ModuleDefinition{}
	err = c.client.Put().
		Resource("moduledefinitions").
		Name(moduleDefinition.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(moduleDefinition).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *moduleDefinitions) UpdateStatus(ctx context.Context, moduleDefinition *v1beta2.ModuleDefinition, opts v1.UpdateOptions) (result *v1beta2.ModuleDefinition, err error) {
	result = &v1beta2.ModuleDefinition{}
	err = c.client.Put().
		Resource("moduledefinitions").
		Name(moduleDefinition.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(moduleDefinition).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the moduleDefinition and deletes it. Returns an error if one occurs.
func (c *moduleDefinitions) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("moduledefinitions").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *moduleDefinitions) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("moduledefinitions").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched moduleDefinition.
func (c *moduleDefinitions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.ModuleDefinition, err error) {
	result = &v1beta2.ModuleDefinition{}
	err = c.client.Patch(pt).
		Resource("moduledefinitions").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
