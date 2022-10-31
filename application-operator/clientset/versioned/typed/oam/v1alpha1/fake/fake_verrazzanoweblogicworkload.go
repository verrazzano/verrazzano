// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVerrazzanoWebLogicWorkloads implements VerrazzanoWebLogicWorkloadInterface
type FakeVerrazzanoWebLogicWorkloads struct {
	Fake *FakeOamV1alpha1
	ns   string
}

var verrazzanoweblogicworkloadsResource = schema.GroupVersionResource{Group: "oam.verrazzano.io", Version: "v1alpha1", Resource: "verrazzanoweblogicworkloads"}

var verrazzanoweblogicworkloadsKind = schema.GroupVersionKind{Group: "oam.verrazzano.io", Version: "v1alpha1", Kind: "VerrazzanoWebLogicWorkload"}

// Get takes name of the verrazzanoWebLogicWorkload, and returns the corresponding verrazzanoWebLogicWorkload object, and an error if there is any.
func (c *FakeVerrazzanoWebLogicWorkloads) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.VerrazzanoWebLogicWorkload, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(verrazzanoweblogicworkloadsResource, c.ns, name), &v1alpha1.VerrazzanoWebLogicWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoWebLogicWorkload), err
}

// List takes label and field selectors, and returns the list of VerrazzanoWebLogicWorkloads that match those selectors.
func (c *FakeVerrazzanoWebLogicWorkloads) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.VerrazzanoWebLogicWorkloadList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(verrazzanoweblogicworkloadsResource, verrazzanoweblogicworkloadsKind, c.ns, opts), &v1alpha1.VerrazzanoWebLogicWorkloadList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.VerrazzanoWebLogicWorkloadList{ListMeta: obj.(*v1alpha1.VerrazzanoWebLogicWorkloadList).ListMeta}
	for _, item := range obj.(*v1alpha1.VerrazzanoWebLogicWorkloadList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested verrazzanoWebLogicWorkloads.
func (c *FakeVerrazzanoWebLogicWorkloads) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(verrazzanoweblogicworkloadsResource, c.ns, opts))

}

// Create takes the representation of a verrazzanoWebLogicWorkload and creates it.  Returns the server's representation of the verrazzanoWebLogicWorkload, and an error, if there is any.
func (c *FakeVerrazzanoWebLogicWorkloads) Create(ctx context.Context, verrazzanoWebLogicWorkload *v1alpha1.VerrazzanoWebLogicWorkload, opts v1.CreateOptions) (result *v1alpha1.VerrazzanoWebLogicWorkload, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(verrazzanoweblogicworkloadsResource, c.ns, verrazzanoWebLogicWorkload), &v1alpha1.VerrazzanoWebLogicWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoWebLogicWorkload), err
}

// Update takes the representation of a verrazzanoWebLogicWorkload and updates it. Returns the server's representation of the verrazzanoWebLogicWorkload, and an error, if there is any.
func (c *FakeVerrazzanoWebLogicWorkloads) Update(ctx context.Context, verrazzanoWebLogicWorkload *v1alpha1.VerrazzanoWebLogicWorkload, opts v1.UpdateOptions) (result *v1alpha1.VerrazzanoWebLogicWorkload, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(verrazzanoweblogicworkloadsResource, c.ns, verrazzanoWebLogicWorkload), &v1alpha1.VerrazzanoWebLogicWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoWebLogicWorkload), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeVerrazzanoWebLogicWorkloads) UpdateStatus(ctx context.Context, verrazzanoWebLogicWorkload *v1alpha1.VerrazzanoWebLogicWorkload, opts v1.UpdateOptions) (*v1alpha1.VerrazzanoWebLogicWorkload, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(verrazzanoweblogicworkloadsResource, "status", c.ns, verrazzanoWebLogicWorkload), &v1alpha1.VerrazzanoWebLogicWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoWebLogicWorkload), err
}

// Delete takes name of the verrazzanoWebLogicWorkload and deletes it. Returns an error if one occurs.
func (c *FakeVerrazzanoWebLogicWorkloads) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(verrazzanoweblogicworkloadsResource, c.ns, name, opts), &v1alpha1.VerrazzanoWebLogicWorkload{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVerrazzanoWebLogicWorkloads) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(verrazzanoweblogicworkloadsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.VerrazzanoWebLogicWorkloadList{})
	return err
}

// Patch applies the patch and returns the patched verrazzanoWebLogicWorkload.
func (c *FakeVerrazzanoWebLogicWorkloads) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.VerrazzanoWebLogicWorkload, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(verrazzanoweblogicworkloadsResource, c.ns, name, pt, data, subresources...), &v1alpha1.VerrazzanoWebLogicWorkload{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.VerrazzanoWebLogicWorkload), err
}
