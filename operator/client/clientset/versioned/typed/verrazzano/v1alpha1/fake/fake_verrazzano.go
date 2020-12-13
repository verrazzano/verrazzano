// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/verrazzano/verrazzano/operator/apis/verrazzano/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVerrazzanos implements VerrazzanoInterface
type FakeVerrazzanos struct {
	Fake *FakeVerrazzanoV1alpha1
	ns   string
}

var verrazzanosResource = schema.GroupVersionResource{Group: "verrazzano", Version: "v1alpha1", Resource: "verrazzanos"}

var verrazzanosKind = schema.GroupVersionKind{Group: "verrazzano", Version: "v1alpha1", Kind: "Verrazzano"}

// Get takes name of the verrazzano, and returns the corresponding verrazzano object, and an error if there is any.
func (c *FakeVerrazzanos) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Verrazzano, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(verrazzanosResource, c.ns, name), &v1alpha1.Verrazzano{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Verrazzano), err
}

// List takes label and field selectors, and returns the list of Verrazzanos that match those selectors.
func (c *FakeVerrazzanos) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.VerrazzanoList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(verrazzanosResource, verrazzanosKind, c.ns, opts), &v1alpha1.VerrazzanoList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.VerrazzanoList{ListMeta: obj.(*v1alpha1.VerrazzanoList).ListMeta}
	for _, item := range obj.(*v1alpha1.VerrazzanoList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested verrazzanos.
func (c *FakeVerrazzanos) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(verrazzanosResource, c.ns, opts))

}

// Create takes the representation of a verrazzano and creates it.  Returns the server's representation of the verrazzano, and an error, if there is any.
func (c *FakeVerrazzanos) Create(ctx context.Context, verrazzano *v1alpha1.Verrazzano, opts v1.CreateOptions) (result *v1alpha1.Verrazzano, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(verrazzanosResource, c.ns, verrazzano), &v1alpha1.Verrazzano{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Verrazzano), err
}

// Update takes the representation of a verrazzano and updates it. Returns the server's representation of the verrazzano, and an error, if there is any.
func (c *FakeVerrazzanos) Update(ctx context.Context, verrazzano *v1alpha1.Verrazzano, opts v1.UpdateOptions) (result *v1alpha1.Verrazzano, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(verrazzanosResource, c.ns, verrazzano), &v1alpha1.Verrazzano{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Verrazzano), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeVerrazzanos) UpdateStatus(ctx context.Context, verrazzano *v1alpha1.Verrazzano, opts v1.UpdateOptions) (*v1alpha1.Verrazzano, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(verrazzanosResource, "status", c.ns, verrazzano), &v1alpha1.Verrazzano{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Verrazzano), err
}

// Delete takes name of the verrazzano and deletes it. Returns an error if one occurs.
func (c *FakeVerrazzanos) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(verrazzanosResource, c.ns, name), &v1alpha1.Verrazzano{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVerrazzanos) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(verrazzanosResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.VerrazzanoList{})
	return err
}

// Patch applies the patch and returns the patched verrazzano.
func (c *FakeVerrazzanos) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Verrazzano, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(verrazzanosResource, c.ns, name, pt, data, subresources...), &v1alpha1.Verrazzano{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Verrazzano), err
}
