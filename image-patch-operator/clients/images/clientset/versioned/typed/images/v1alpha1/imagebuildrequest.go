// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/verrazzano/verrazzano/image-patch-operator/api/images/v1alpha1"
	scheme "github.com/verrazzano/verrazzano/image-patch-operator/clients/images/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ImageBuildRequestsGetter has a method to return a ImageBuildRequestInterface.
// A group's client should implement this interface.
type ImageBuildRequestsGetter interface {
	ImageBuildRequests(namespace string) ImageBuildRequestInterface
}

// ImageBuildRequestInterface has methods to work with ImageBuildRequest resources.
type ImageBuildRequestInterface interface {
	Create(ctx context.Context, imageBuildRequest *v1alpha1.ImageBuildRequest, opts v1.CreateOptions) (*v1alpha1.ImageBuildRequest, error)
	Update(ctx context.Context, imageBuildRequest *v1alpha1.ImageBuildRequest, opts v1.UpdateOptions) (*v1alpha1.ImageBuildRequest, error)
	UpdateStatus(ctx context.Context, imageBuildRequest *v1alpha1.ImageBuildRequest, opts v1.UpdateOptions) (*v1alpha1.ImageBuildRequest, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ImageBuildRequest, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ImageBuildRequestList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ImageBuildRequest, err error)
	ImageBuildRequestExpansion
}

// imageBuildRequests implements ImageBuildRequestInterface
type imageBuildRequests struct {
	client rest.Interface
	ns     string
}

// newImageBuildRequests returns a ImageBuildRequests
func newImageBuildRequests(c *ImagesV1alpha1Client, namespace string) *imageBuildRequests {
	return &imageBuildRequests{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the imageBuildRequest, and returns the corresponding imageBuildRequest object, and an error if there is any.
func (c *imageBuildRequests) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ImageBuildRequest, err error) {
	result = &v1alpha1.ImageBuildRequest{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("imagebuildrequests").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ImageBuildRequests that match those selectors.
func (c *imageBuildRequests) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ImageBuildRequestList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ImageBuildRequestList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("imagebuildrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested imageBuildRequests.
func (c *imageBuildRequests) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("imagebuildrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a imageBuildRequest and creates it.  Returns the server's representation of the imageBuildRequest, and an error, if there is any.
func (c *imageBuildRequests) Create(ctx context.Context, imageBuildRequest *v1alpha1.ImageBuildRequest, opts v1.CreateOptions) (result *v1alpha1.ImageBuildRequest, err error) {
	result = &v1alpha1.ImageBuildRequest{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("imagebuildrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(imageBuildRequest).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a imageBuildRequest and updates it. Returns the server's representation of the imageBuildRequest, and an error, if there is any.
func (c *imageBuildRequests) Update(ctx context.Context, imageBuildRequest *v1alpha1.ImageBuildRequest, opts v1.UpdateOptions) (result *v1alpha1.ImageBuildRequest, err error) {
	result = &v1alpha1.ImageBuildRequest{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("imagebuildrequests").
		Name(imageBuildRequest.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(imageBuildRequest).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *imageBuildRequests) UpdateStatus(ctx context.Context, imageBuildRequest *v1alpha1.ImageBuildRequest, opts v1.UpdateOptions) (result *v1alpha1.ImageBuildRequest, err error) {
	result = &v1alpha1.ImageBuildRequest{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("imagebuildrequests").
		Name(imageBuildRequest.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(imageBuildRequest).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the imageBuildRequest and deletes it. Returns an error if one occurs.
func (c *imageBuildRequests) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("imagebuildrequests").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *imageBuildRequests) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("imagebuildrequests").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched imageBuildRequest.
func (c *imageBuildRequests) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ImageBuildRequest, err error) {
	result = &v1alpha1.ImageBuildRequest{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("imagebuildrequests").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
