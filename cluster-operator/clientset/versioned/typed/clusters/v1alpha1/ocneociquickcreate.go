// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	scheme "github.com/verrazzano/verrazzano/cluster-operator/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// OCNEOCIQuickCreatesGetter has a method to return a OCNEOCIQuickCreateInterface.
// A group's client should implement this interface.
type OCNEOCIQuickCreatesGetter interface {
	OCNEOCIQuickCreates(namespace string) OCNEOCIQuickCreateInterface
}

// OCNEOCIQuickCreateInterface has methods to work with OCNEOCIQuickCreate resources.
type OCNEOCIQuickCreateInterface interface {
	Create(ctx context.Context, oCNEOCIQuickCreate *v1alpha1.OCNEOCIQuickCreate, opts v1.CreateOptions) (*v1alpha1.OCNEOCIQuickCreate, error)
	Update(ctx context.Context, oCNEOCIQuickCreate *v1alpha1.OCNEOCIQuickCreate, opts v1.UpdateOptions) (*v1alpha1.OCNEOCIQuickCreate, error)
	UpdateStatus(ctx context.Context, oCNEOCIQuickCreate *v1alpha1.OCNEOCIQuickCreate, opts v1.UpdateOptions) (*v1alpha1.OCNEOCIQuickCreate, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.OCNEOCIQuickCreate, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.OCNEOCIQuickCreateList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.OCNEOCIQuickCreate, err error)
	OCNEOCIQuickCreateExpansion
}

// oCNEOCIQuickCreates implements OCNEOCIQuickCreateInterface
type oCNEOCIQuickCreates struct {
	client rest.Interface
	ns     string
}

// newOCNEOCIQuickCreates returns a OCNEOCIQuickCreates
func newOCNEOCIQuickCreates(c *ClustersV1alpha1Client, namespace string) *oCNEOCIQuickCreates {
	return &oCNEOCIQuickCreates{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the oCNEOCIQuickCreate, and returns the corresponding oCNEOCIQuickCreate object, and an error if there is any.
func (c *oCNEOCIQuickCreates) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.OCNEOCIQuickCreate, err error) {
	result = &v1alpha1.OCNEOCIQuickCreate{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ocneociquickcreates").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of OCNEOCIQuickCreates that match those selectors.
func (c *oCNEOCIQuickCreates) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.OCNEOCIQuickCreateList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.OCNEOCIQuickCreateList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ocneociquickcreates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested oCNEOCIQuickCreates.
func (c *oCNEOCIQuickCreates) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("ocneociquickcreates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a oCNEOCIQuickCreate and creates it.  Returns the server's representation of the oCNEOCIQuickCreate, and an error, if there is any.
func (c *oCNEOCIQuickCreates) Create(ctx context.Context, oCNEOCIQuickCreate *v1alpha1.OCNEOCIQuickCreate, opts v1.CreateOptions) (result *v1alpha1.OCNEOCIQuickCreate, err error) {
	result = &v1alpha1.OCNEOCIQuickCreate{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("ocneociquickcreates").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(oCNEOCIQuickCreate).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a oCNEOCIQuickCreate and updates it. Returns the server's representation of the oCNEOCIQuickCreate, and an error, if there is any.
func (c *oCNEOCIQuickCreates) Update(ctx context.Context, oCNEOCIQuickCreate *v1alpha1.OCNEOCIQuickCreate, opts v1.UpdateOptions) (result *v1alpha1.OCNEOCIQuickCreate, err error) {
	result = &v1alpha1.OCNEOCIQuickCreate{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("ocneociquickcreates").
		Name(oCNEOCIQuickCreate.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(oCNEOCIQuickCreate).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *oCNEOCIQuickCreates) UpdateStatus(ctx context.Context, oCNEOCIQuickCreate *v1alpha1.OCNEOCIQuickCreate, opts v1.UpdateOptions) (result *v1alpha1.OCNEOCIQuickCreate, err error) {
	result = &v1alpha1.OCNEOCIQuickCreate{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("ocneociquickcreates").
		Name(oCNEOCIQuickCreate.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(oCNEOCIQuickCreate).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the oCNEOCIQuickCreate and deletes it. Returns an error if one occurs.
func (c *oCNEOCIQuickCreates) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ocneociquickcreates").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *oCNEOCIQuickCreates) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ocneociquickcreates").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched oCNEOCIQuickCreate.
func (c *oCNEOCIQuickCreates) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.OCNEOCIQuickCreate, err error) {
	result = &v1alpha1.OCNEOCIQuickCreate{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("ocneociquickcreates").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
