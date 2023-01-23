// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	scheme "github.com/verrazzano/verrazzano/application-operator/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// IngressTraitsGetter has a method to return a IngressTraitInterface.
// A group's client should implement this interface.
type IngressTraitsGetter interface {
	IngressTraits(namespace string) IngressTraitInterface
}

// IngressTraitInterface has methods to work with IngressTrait resources.
type IngressTraitInterface interface {
	Create(ctx context.Context, ingressTrait *v1alpha1.IngressTrait, opts v1.CreateOptions) (*v1alpha1.IngressTrait, error)
	Update(ctx context.Context, ingressTrait *v1alpha1.IngressTrait, opts v1.UpdateOptions) (*v1alpha1.IngressTrait, error)
	UpdateStatus(ctx context.Context, ingressTrait *v1alpha1.IngressTrait, opts v1.UpdateOptions) (*v1alpha1.IngressTrait, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.IngressTrait, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.IngressTraitList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.IngressTrait, err error)
	IngressTraitExpansion
}

// ingressTraits implements IngressTraitInterface
type ingressTraits struct {
	client rest.Interface
	ns     string
}

// newIngressTraits returns a IngressTraits
func newIngressTraits(c *OamV1alpha1Client, namespace string) *ingressTraits {
	return &ingressTraits{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the ingressTrait, and returns the corresponding ingressTrait object, and an error if there is any.
func (c *ingressTraits) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.IngressTrait, err error) {
	result = &v1alpha1.IngressTrait{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ingresstraits").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of IngressTraits that match those selectors.
func (c *ingressTraits) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.IngressTraitList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.IngressTraitList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("ingresstraits").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested ingressTraits.
func (c *ingressTraits) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("ingresstraits").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a ingressTrait and creates it.  Returns the server's representation of the ingressTrait, and an error, if there is any.
func (c *ingressTraits) Create(ctx context.Context, ingressTrait *v1alpha1.IngressTrait, opts v1.CreateOptions) (result *v1alpha1.IngressTrait, err error) {
	result = &v1alpha1.IngressTrait{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("ingresstraits").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ingressTrait).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a ingressTrait and updates it. Returns the server's representation of the ingressTrait, and an error, if there is any.
func (c *ingressTraits) Update(ctx context.Context, ingressTrait *v1alpha1.IngressTrait, opts v1.UpdateOptions) (result *v1alpha1.IngressTrait, err error) {
	result = &v1alpha1.IngressTrait{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("ingresstraits").
		Name(ingressTrait.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ingressTrait).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *ingressTraits) UpdateStatus(ctx context.Context, ingressTrait *v1alpha1.IngressTrait, opts v1.UpdateOptions) (result *v1alpha1.IngressTrait, err error) {
	result = &v1alpha1.IngressTrait{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("ingresstraits").
		Name(ingressTrait.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ingressTrait).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the ingressTrait and deletes it. Returns an error if one occurs.
func (c *ingressTraits) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ingresstraits").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *ingressTraits) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("ingresstraits").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched ingressTrait.
func (c *ingressTraits) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.IngressTrait, err error) {
	result = &v1alpha1.IngressTrait{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("ingresstraits").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
